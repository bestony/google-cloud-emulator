package gcs

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bestony/google-cloud-emulator/core/auth"
	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	"github.com/bestony/google-cloud-emulator/core/iam"
)

// Handler serves the Cloud Storage JSON API under /storage/v1/.
// Routing is hand-rolled because the JSON API has no protobuf surface.
type Handler struct {
	state *State
	now   func() time.Time
}

// NewHandler returns a Handler backed by state.
func NewHandler(state *State) *Handler {
	return &Handler{state: state, now: func() time.Time { return time.Now().UTC() }}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code gcperr.Code, format string, args ...any) {
	gcperr.WriteREST(w, code, format, args...)
}

// ServeHTTP routes /storage/v1/... and /upload/storage/v1/... requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	escapedPath := r.URL.EscapedPath()
	var segs []string
	upload := false
	if strings.HasPrefix(escapedPath, "/storage/v1/") {
		segs = splitEscapedPath(strings.TrimPrefix(escapedPath, "/storage/v1/"))
	} else if strings.HasPrefix(escapedPath, "/upload/storage/v1/") {
		segs = splitEscapedPath(strings.TrimPrefix(escapedPath, "/upload/storage/v1/"))
		upload = true
	} else {
		writeError(w, gcperr.NotFound, "unknown path %q", r.URL.Path)
		return
	}
	if len(segs) == 0 || segs[0] == "" {
		writeError(w, gcperr.NotFound, "unknown path %q", r.URL.Path)
		return
	}

	if upload {
		h.routeUpload(w, r, segs)
		return
	}
	switch segs[0] {
	case "b":
		h.routeBuckets(w, r, segs[1:])
	default:
		writeError(w, gcperr.NotFound, "unknown path %q", r.URL.Path)
	}
}

// splitEscapedPath splits before unescaping each segment, so an object name
// containing a slash can be addressed as a single %2F-escaped path segment.
func splitEscapedPath(path string) []string {
	raw := strings.Split(path, "/")
	for i, seg := range raw {
		if decoded, err := url.PathUnescape(seg); err == nil {
			raw[i] = decoded
		}
	}
	return raw
}

func (h *Handler) routeBuckets(w http.ResponseWriter, r *http.Request, segs []string) {
	if len(segs) == 0 {
		switch r.Method {
		case http.MethodGet:
			h.bucketsList(w, r)
		case http.MethodPost:
			h.bucketInsert(w, r)
		default:
			writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
		}
		return
	}
	name := segs[0]
	if len(segs) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.bucketGet(w, r, name)
		case http.MethodPatch:
			h.bucketPatch(w, r, name)
		case http.MethodPut:
			h.bucketUpdate(w, r, name)
		case http.MethodDelete:
			h.bucketDelete(w, r, name)
		default:
			writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
		}
		return
	}
	if segs[1] == "iam" {
		h.bucketIAM(w, r, name, segs[2:])
		return
	}
	if segs[1] == "o" {
		h.routeObjects(w, r, name, segs[2:])
		return
	}
	writeError(w, gcperr.NotFound, "unknown path %q", r.URL.Path)
}

// --- bucket CRUD ---

var bucketNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]{1,61}[a-z0-9]$`)

func validBucketName(name string) bool {
	if !bucketNameRe.MatchString(name) || strings.Contains(name, "..") {
		return false
	}
	// Reject names that look like an IPv4 address.
	parts := strings.Split(name, ".")
	if len(parts) == 4 {
		numeric := true
		for _, p := range parts {
			if _, err := strconv.Atoi(p); err != nil {
				numeric = false
				break
			}
		}
		if numeric {
			return false
		}
	}
	return true
}

// serverManagedKeys are recomputed by the emulator and cannot be set by
// clients on insert/patch/update.
var serverManagedKeys = map[string]bool{
	"kind": true, "id": true, "selfLink": true, "name": true,
	"projectNumber": true, "timeCreated": true, "updated": true,
	"metageneration": true, "etag": true, "locationType": true,
}

func bucketSelfLink(name string) string {
	return "https://storage.googleapis.com/storage/v1/b/" + name
}

func bucketETag(b map[string]any) string {
	raw, _ := json.Marshal(b)
	sum := sha1.Sum(raw)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func (h *Handler) refreshMeta(b map[string]any, name, project, timeCreated string, metaGen int64) {
	b["kind"] = "storage#bucket"
	b["id"] = name
	b["selfLink"] = bucketSelfLink(name)
	b["name"] = name
	b["project"] = project
	b["timeCreated"] = timeCreated
	b["updated"] = h.now().Format(time.RFC3339)
	b["metageneration"] = strconv.FormatInt(metaGen, 10)
	if _, ok := b["location"]; !ok {
		b["location"] = "US"
	}
	if _, ok := b["storageClass"]; !ok {
		b["storageClass"] = "STANDARD"
	}
	if _, ok := b["iamConfiguration"]; !ok {
		b["iamConfiguration"] = map[string]any{
			"bucketPolicyOnly":         map[string]any{"enabled": false},
			"uniformBucketLevelAccess": map[string]any{"enabled": false},
			"publicAccessPrevention":   "inherited",
		}
	}
	delete(b, "etag")
	b["etag"] = bucketETag(b)
}

func (h *Handler) bucketsList(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		writeError(w, gcperr.InvalidArgument, "project is required")
		return
	}
	prefix := r.URL.Query().Get("prefix")
	maxResults := 1000
	if v := r.URL.Query().Get("maxResults"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 1000 {
			writeError(w, gcperr.InvalidArgument, "invalid maxResults %q", v)
			return
		}
		maxResults = n
	}
	pageToken := r.URL.Query().Get("pageToken")

	all := h.state.list()
	names := make([]string, 0, len(all))
	for _, b := range all {
		name, _ := b["name"].(string)
		if b["project"] == project && strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	start := 0
	if pageToken != "" {
		n, err := strconv.Atoi(pageToken)
		if err != nil {
			writeError(w, gcperr.InvalidArgument, "invalid pageToken %q", pageToken)
			return
		}
		start = n
	}
	if start > len(names) {
		start = len(names)
	}
	end := start + maxResults
	if end > len(names) {
		end = len(names)
	}
	items := make([]map[string]any, 0, end-start)
	for _, n := range names[start:end] {
		if b, ok := h.state.get(n); ok {
			items = append(items, b)
		}
	}
	resp := map[string]any{"kind": "storage#buckets", "items": items}
	if end < len(names) {
		resp["nextPageToken"] = strconv.Itoa(end)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) readBucketBody(r *http.Request) (map[string]any, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (h *Handler) bucketInsert(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		writeError(w, gcperr.InvalidArgument, "project is required")
		return
	}
	m, err := h.readBucketBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON body: %v", err)
		return
	}
	name, _ := m["name"].(string)
	if name == "" {
		writeError(w, gcperr.InvalidArgument, "bucket name is required")
		return
	}
	if !validBucketName(name) {
		writeError(w, gcperr.InvalidArgument, "invalid bucket name %q", name)
		return
	}
	if _, exists := h.state.get(name); exists {
		writeError(w, gcperr.AlreadyExists, "bucket %q already exists", name)
		return
	}
	stored := map[string]any{}
	for k, v := range m {
		if !serverManagedKeys[k] {
			stored[k] = v
		}
	}
	now := h.now().Format(time.RFC3339)
	h.refreshMeta(stored, name, project, now, 1)
	h.state.put(name, stored)
	writeJSON(w, http.StatusOK, stored)
}

func (h *Handler) bucketGet(w http.ResponseWriter, r *http.Request, name string) {
	b, ok := h.state.get(name)
	if !ok {
		writeError(w, gcperr.NotFound, "bucket %q not found", name)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// mergePatch applies JSON Merge Patch (RFC 7386) semantics: objects merge
// recursively, null deletes a key, everything else replaces.
func mergePatch(target, patch map[string]any) {
	for k, v := range patch {
		if v == nil {
			delete(target, k)
			continue
		}
		if patchObj, ok := v.(map[string]any); ok {
			if targetObj, ok := target[k].(map[string]any); ok {
				mergePatch(targetObj, patchObj)
				continue
			}
		}
		target[k] = v
	}
}

func currentMetaGen(b map[string]any) int64 {
	if s, ok := b["metageneration"].(string); ok {
		n, _ := strconv.ParseInt(s, 10, 64)
		return n
	}
	return 1
}

func (h *Handler) bucketPatch(w http.ResponseWriter, r *http.Request, name string) {
	stored, ok := h.state.get(name)
	if !ok {
		writeError(w, gcperr.NotFound, "bucket %q not found", name)
		return
	}
	m, err := h.readBucketBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON body: %v", err)
		return
	}
	for k := range m {
		if serverManagedKeys[k] {
			delete(m, k)
		}
	}
	mergePatch(stored, m)
	h.refreshMeta(stored, name, fmt.Sprint(stored["project"]),
		fmt.Sprint(stored["timeCreated"]), currentMetaGen(stored)+1)
	h.state.put(name, stored)
	writeJSON(w, http.StatusOK, stored)
}

// bucketUpdate implements PUT (full replace, preserving server fields).
func (h *Handler) bucketUpdate(w http.ResponseWriter, r *http.Request, name string) {
	old, ok := h.state.get(name)
	if !ok {
		writeError(w, gcperr.NotFound, "bucket %q not found", name)
		return
	}
	m, err := h.readBucketBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON body: %v", err)
		return
	}
	stored := map[string]any{}
	for k, v := range m {
		if !serverManagedKeys[k] {
			stored[k] = v
		}
	}
	h.refreshMeta(stored, name, fmt.Sprint(old["project"]),
		fmt.Sprint(old["timeCreated"]), currentMetaGen(old)+1)
	h.state.put(name, stored)
	writeJSON(w, http.StatusOK, stored)
}

func (h *Handler) bucketDelete(w http.ResponseWriter, r *http.Request, name string) {
	if _, ok := h.state.get(name); !ok {
		writeError(w, gcperr.NotFound, "bucket %q not found", name)
		return
	}
	h.state.delete(name)
	writeJSON(w, http.StatusOK, nil)
}

// --- bucket IAM ---

func (h *Handler) bucketIAM(w http.ResponseWriter, r *http.Request, name string, segs []string) {
	if _, ok := h.state.get(name); !ok {
		writeError(w, gcperr.NotFound, "bucket %q not found", name)
		return
	}
	key := iamKey(name)
	if len(segs) == 1 && segs[0] == "test" {
		if r.Method != http.MethodPost {
			writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
			return
		}
		h.iamTest(w, r, key, name)
		return
	}
	if len(segs) != 0 {
		writeError(w, gcperr.NotFound, "unknown path %q", r.URL.Path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, policyToJSON(h.state.IAM().Get(key), name))
	case http.MethodPut, http.MethodPost:
		h.iamSet(w, r, key, name)
	case http.MethodDelete:
		h.state.IAM().Set(key, iam.Policy{Version: 3})
		writeJSON(w, http.StatusOK, policyToJSON(h.state.IAM().Get(key), name))
	default:
		writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
	}
}

func policyToJSON(p iam.Policy, bucket string) map[string]any {
	bindings := make([]map[string]any, 0, len(p.Bindings))
	for _, b := range p.Bindings {
		bindings = append(bindings, map[string]any{"role": b.Role, "members": b.Members})
	}
	return map[string]any{
		"kind":     "storage#policy",
		"version":  p.Version,
		"etag":     p.Etag,
		"bindings": bindings,
		"resourceId": map[string]any{
			"project":     "_",
			"type":        "storage#bucket",
			"bucket_name": bucket,
		},
	}
}

func (h *Handler) iamSet(w http.ResponseWriter, r *http.Request, key, bucket string) {
	m, err := h.readBucketBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON body: %v", err)
		return
	}
	pol, _ := m["policy"].(map[string]any)
	if pol == nil {
		pol = m // clients may send the Policy object directly
	}
	p := iam.Policy{Etag: ""}
	if v, ok := pol["version"].(float64); ok {
		p.Version = int32(v)
	}
	if raw, ok := pol["bindings"].([]any); ok {
		for _, item := range raw {
			bm, _ := item.(map[string]any)
			if bm == nil {
				continue
			}
			role, _ := bm["role"].(string)
			members := []string{}
			if ms, ok := bm["members"].([]any); ok {
				for _, mv := range ms {
					if s, ok := mv.(string); ok {
						members = append(members, s)
					}
				}
			}
			p.Bindings = append(p.Bindings, iam.Binding{Role: role, Members: members})
		}
	}
	h.state.IAM().Set(key, p)
	writeJSON(w, http.StatusOK, policyToJSON(h.state.IAM().Get(key), bucket))
}

func (h *Handler) iamTest(w http.ResponseWriter, r *http.Request, key, bucket string) {
	m, err := h.readBucketBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON body: %v", err)
		return
	}
	raw, _ := m["permissions"].([]any)
	perms := make([]string, 0, len(raw))
	for _, p := range raw {
		if s, ok := p.(string); ok {
			perms = append(perms, s)
		}
	}
	got := h.state.IAM().Test(key, perms, auth.FromRequest(r).Email)
	if got == nil {
		got = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"kind": "storage#testIamPermissionsResponse", "permissions": got})
}
