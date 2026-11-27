package dns

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bestony/google-cloud-emulator/core/auth"
	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	"github.com/bestony/google-cloud-emulator/core/iam"
)

// defaultPageSize is used when maxResults is not specified.
const defaultPageSize = 100

// Handler serves the Cloud DNS REST surface under /dns/v1/.
type Handler struct {
	state *State
}

// NewHandler returns a Handler backed by state.
func NewHandler(state *State) *Handler { return &Handler{state: state} }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		_, _ = w.Write([]byte("{}"))
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code gcperr.Code, format string, args ...any) {
	gcperr.WriteREST(w, code, format, args...)
}

// ServeHTTP routes /dns/v1/... requests. Routing is hand-rolled because the
// API's {+name} wildcards and ":verb" suffixes do not fit net/http patterns.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/dns/v1/") {
		writeError(w, gcperr.NotFound, "unknown path %q", path)
		return
	}
	segs := strings.Split(strings.TrimPrefix(path, "/dns/v1/"), "/")

	if isUnimplementedFamily(segs) {
		writeError(w, gcperr.Unimplemented, "the %s API surface is not implemented by the emulator", familyName(segs))
		return
	}
	// projects.get: GET /dns/v1/projects/{project}
	if len(segs) == 2 && segs[0] == "projects" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"kind": "dns#project",
			"id":   "projects/" + segs[1],
		})
		return
	}
	if len(segs) >= 3 && segs[0] == "projects" && segs[1] != "" {
		h.routeProject(w, r, segs[1], segs[2:])
		return
	}
	writeError(w, gcperr.NotFound, "unknown path %q", path)
}

var unimplementedFamilies = map[string]bool{
	"dnsKeys": true, "responsePolicies": true, "responsePolicyRules": true,
	"policies": true, "outboundEndpoints": true, "locations": true,
}

func isUnimplementedFamily(segs []string) bool {
	for _, s := range segs {
		if unimplementedFamilies[s] {
			return true
		}
	}
	return false
}

func familyName(segs []string) string {
	for _, s := range segs {
		if unimplementedFamilies[s] {
			return s
		}
	}
	return "requested"
}

func (h *Handler) routeProject(w http.ResponseWriter, r *http.Request, project string, segs []string) {
	switch segs[0] {
	case "managedZones":
		h.routeManagedZones(w, r, project, segs[1:])
	case "operations":
		h.routeOperations(w, r, project, segs[1:])
	default:
		writeError(w, gcperr.NotFound, "unknown path %q", r.URL.Path)
	}
}

// routeManagedZones handles everything under projects/{p}/managedZones.
func (h *Handler) routeManagedZones(w http.ResponseWriter, r *http.Request, project string, segs []string) {
	if len(segs) == 0 {
		switch r.Method {
		case http.MethodPost:
			h.zonesCreate(w, r, project)
		case http.MethodGet:
			h.zonesList(w, r, project)
		default:
			methodNotAllowed(w, r)
		}
		return
	}
	zone, verb, isVerb := splitVerb(segs[0])
	if isVerb {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, r)
			return
		}
		h.zoneIAM(w, r, project, zone, verb)
		return
	}
	if len(segs) == 1 {
		h.zoneItem(w, r, project, zone)
		return
	}
	switch segs[1] {
	case "changes":
		h.routeChanges(w, r, project, zone, segs[2:])
	case "rrsets":
		h.routeRRSets(w, r, project, zone, segs[2:])
	case "operations":
		h.routeZoneOperations(w, r, project, zone, segs[2:])
	default:
		writeError(w, gcperr.NotFound, "unknown path %q", r.URL.Path)
	}
}

// splitVerb splits "zone:getIamPolicy" into zone and verb.
func splitVerb(seg string) (zone, verb string, ok bool) {
	if i := strings.Index(seg, ":"); i >= 0 {
		return seg[:i], seg[i+1:], true
	}
	return seg, "", false
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, gcperr.InvalidArgument, "method %s is not supported for %q", r.Method, r.URL.Path)
}

// --- IAM ---

func (h *Handler) zoneIAM(w http.ResponseWriter, r *http.Request, project, zone, verb string) {
	key := ZoneKey(project, zone)
	if !h.state.ZoneExists(project, zone) {
		writeError(w, gcperr.NotFound, "The resource '%s' was not found", key)
		return
	}
	store := h.state.IAMStore()
	switch verb {
	case "getIamPolicy":
		p := store.Get(key)
		writeJSON(w, http.StatusOK, map[string]any{
			"version":  p.Version,
			"bindings": toJSONBindings(p.Bindings),
			"etag":     p.Etag,
		})
	case "setIamPolicy":
		var body struct {
			Policy *struct {
				Version  int32 `json:"version"`
				Bindings []struct {
					Role      string          `json:"role"`
					Members   []string        `json:"members"`
					Condition json.RawMessage `json:"condition"`
				} `json:"bindings"`
			} `json:"policy"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		if body.Policy == nil {
			writeError(w, gcperr.InvalidArgument, "policy is required")
			return
		}
		bindings := make([]iam.Binding, 0, len(body.Policy.Bindings))
		for _, b := range body.Policy.Bindings {
			if len(b.Condition) > 0 && string(b.Condition) != "null" {
				writeError(w, gcperr.Unimplemented, "IAM conditions are not supported by the emulator")
				return
			}
			if b.Role == "" {
				writeError(w, gcperr.InvalidArgument, "binding role is required")
				return
			}
			bindings = append(bindings, iam.Binding{Role: b.Role, Members: append([]string(nil), b.Members...)})
		}
		version := body.Policy.Version
		if version == 0 {
			version = 1
		}
		saved := store.Set(key, iam.Policy{Version: version, Bindings: bindings})
		writeJSON(w, http.StatusOK, map[string]any{
			"version":  saved.Version,
			"bindings": toJSONBindings(saved.Bindings),
			"etag":     saved.Etag,
		})
	case "testIamPermissions":
		var body struct {
			Permissions []string `json:"permissions"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		caller := auth.FromRequest(r)
		allowed := store.Test(key, body.Permissions, caller.Email)
		writeJSON(w, http.StatusOK, map[string]any{"permissions": allowed})
	default:
		writeError(w, gcperr.NotFound, "unknown method %q", verb)
	}
}

// --- projects ---

// (projects.get is handled directly in ServeHTTP.)

// --- managed zones ---

func (h *Handler) zonesCreate(w http.ResponseWriter, r *http.Request, project string) {
	var body map[string]any
	if !decodeBodyMap(w, r, &body) {
		return
	}
	name, _ := body["name"].(string)
	if name == "" {
		writeError(w, gcperr.InvalidArgument, "name is required")
		return
	}
	if !ValidateZoneName(name) {
		writeError(w, gcperr.InvalidArgument, "invalid zone name %q", name)
		return
	}
	dnsName, _ := body["dnsName"].(string)
	if strings.TrimSpace(dnsName) == "" {
		writeError(w, gcperr.InvalidArgument, "dnsName is required")
		return
	}
	zone, code := h.state.CreateZone(project, name, body)
	if code == "already-exists" {
		writeError(w, gcperr.AlreadyExists, "The resource '%s' already exists", ZoneKey(project, name))
		return
	}
	writeJSON(w, http.StatusOK, zone)
}

func (h *Handler) zonesList(w http.ResponseWriter, r *http.Request, project string) {
	q := r.URL.Query()
	pageSize, ok := parseMaxResults(w, q.Get("maxResults"))
	if !ok {
		return
	}
	zones, next, _, code := h.state.ListZones(project, q.Get("dnsName"), pageSize, q.Get("pageToken"))
	if code == "invalid-page-token" {
		writeError(w, gcperr.InvalidArgument, "invalid page token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind":          "dns#managedZonesListResponse",
		"managedZones":  zones,
		"nextPageToken": next,
	})
}

func (h *Handler) zoneItem(w http.ResponseWriter, r *http.Request, project, zone string) {
	key := ZoneKey(project, zone)
	switch r.Method {
	case http.MethodGet:
		z := h.state.GetZone(project, zone)
		if z == nil {
			writeError(w, gcperr.NotFound, "The resource '%s' was not found", key)
			return
		}
		writeJSON(w, http.StatusOK, z)
	case http.MethodPatch, http.MethodPut:
		var body map[string]any
		if !decodeBodyMap(w, r, &body) {
			return
		}
		z, code := h.state.PatchZone(project, zone, body)
		if code == "not-found" {
			writeError(w, gcperr.NotFound, "The resource '%s' was not found", key)
			return
		}
		writeJSON(w, http.StatusOK, z)
	case http.MethodDelete:
		switch h.state.DeleteZone(project, zone) {
		case "":
			writeJSON(w, http.StatusOK, nil)
		case "not-found":
			writeError(w, gcperr.NotFound, "The resource '%s' was not found", key)
		default:
			writeError(w, gcperr.FailedPrecondition, "The managed zone '%s' still contains record sets; delete them first", key)
		}
	default:
		methodNotAllowed(w, r)
	}
}

// --- changes ---

func (h *Handler) routeChanges(w http.ResponseWriter, r *http.Request, project, zone string, segs []string) {
	if len(segs) == 0 {
		switch r.Method {
		case http.MethodPost:
			h.changesCreate(w, r, project, zone)
		case http.MethodGet:
			h.changesList(w, r, project, zone)
		default:
			methodNotAllowed(w, r)
		}
		return
	}
	if len(segs) == 1 && r.Method == http.MethodGet {
		h.changesGet(w, project, zone, segs[0])
		return
	}
	writeError(w, gcperr.NotFound, "unknown path %q", r.URL.Path)
}

func (h *Handler) changesCreate(w http.ResponseWriter, r *http.Request, project, zone string) {
	var body struct {
		Additions []*rrset `json:"additions"`
		Deletions []*rrset `json:"deletions"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	z := h.state.GetZone(project, zone)
	if z == nil {
		writeError(w, gcperr.NotFound, "The resource '%s' was not found", ZoneKey(project, zone))
		return
	}
	dnsName, _ := z["dnsName"].(string)
	for _, set := range append(append([]*rrset{}, body.Additions...), body.Deletions...) {
		if msg := ValidateRRSet(set, dnsName); msg != "" {
			writeError(w, gcperr.InvalidArgument, "%s", msg)
			return
		}
	}
	for _, a := range body.Additions {
		if len(a.Rrdatas) == 0 {
			writeError(w, gcperr.InvalidArgument, "rrdatas must not be empty for additions")
			return
		}
	}
	chg, code := h.state.CreateChange(project, zone, body.Additions, body.Deletions)
	switch code {
	case "":
		writeJSON(w, http.StatusOK, chg)
	case "zone-not-found":
		writeError(w, gcperr.NotFound, "The resource '%s' was not found", ZoneKey(project, zone))
	case "rrset-not-found":
		writeError(w, gcperr.InvalidArgument, "cannot delete a record set that does not exist")
	default:
		writeError(w, gcperr.InvalidArgument, "conflicting record sets in change")
	}
}

func (h *Handler) changesGet(w http.ResponseWriter, project, zone, id string) {
	if !h.state.ZoneExists(project, zone) {
		writeError(w, gcperr.NotFound, "The resource '%s' was not found", ZoneKey(project, zone))
		return
	}
	chg, ok := h.state.GetChange(project, zone, id)
	if !ok {
		writeError(w, gcperr.NotFound, "The resource 'changes/%s' was not found", id)
		return
	}
	writeJSON(w, http.StatusOK, chg)
}

func (h *Handler) changesList(w http.ResponseWriter, r *http.Request, project, zone string) {
	q := r.URL.Query()
	pageSize, ok := parseMaxResults(w, q.Get("maxResults"))
	if !ok {
		return
	}
	changes, next, _, code := h.state.ListChanges(project, zone, pageSize, q.Get("pageToken"))
	if code == "invalid-page-token" {
		writeError(w, gcperr.InvalidArgument, "invalid page token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind":          "dns#changesListResponse",
		"changes":       changes,
		"nextPageToken": next,
	})
}

// --- resource record sets ---

func (h *Handler) routeRRSets(w http.ResponseWriter, r *http.Request, project, zone string, segs []string) {
	if len(segs) == 0 {
		if r.Method == http.MethodGet {
			h.rrsetsList(w, r, project, zone)
			return
		}
		// Record sets are modified through changes, not directly.
		writeError(w, gcperr.Unimplemented, "direct record set mutation is not supported; use changes.create")
		return
	}
	if len(segs) == 2 && r.Method == http.MethodGet {
		h.rrsetsGet(w, project, zone, segs[0], segs[1])
		return
	}
	writeError(w, gcperr.NotFound, "unknown path %q", r.URL.Path)
}

func (h *Handler) rrsetsGet(w http.ResponseWriter, project, zone, name, typ string) {
	if !h.state.ZoneExists(project, zone) {
		writeError(w, gcperr.NotFound, "The resource '%s' was not found", ZoneKey(project, zone))
		return
	}
	rr, ok := h.state.GetRRSet(project, zone, name, typ)
	if !ok {
		writeError(w, gcperr.NotFound, "The resource record set '%s' of type '%s' was not found", name, typ)
		return
	}
	writeJSON(w, http.StatusOK, rr)
}

func (h *Handler) rrsetsList(w http.ResponseWriter, r *http.Request, project, zone string) {
	q := r.URL.Query()
	if q.Get("filter") != "" {
		writeError(w, gcperr.InvalidArgument, "filter is not supported by the emulator")
		return
	}
	pageSize, ok := parseMaxResults(w, q.Get("maxResults"))
	if !ok {
		return
	}
	rrsets, next, _, code := h.state.ListRRSets(project, zone, q.Get("name"), q.Get("type"), pageSize, q.Get("pageToken"))
	if code == "invalid-page-token" {
		writeError(w, gcperr.InvalidArgument, "invalid page token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind":          "dns#resourceRecordSetsListResponse",
		"rrsets":        rrsets,
		"nextPageToken": next,
	})
}

// --- operations ---

func (h *Handler) routeZoneOperations(w http.ResponseWriter, r *http.Request, project, zone string, segs []string) {
	// The emulator performs every mutation synchronously, so no managed zone
	// operations are ever created.
	if len(segs) == 0 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"kind":       "dns#managedZoneOperationsListResponse",
			"operations": []any{},
		})
		return
	}
	writeError(w, gcperr.NotFound, "operation not found")
}

func (h *Handler) routeOperations(w http.ResponseWriter, r *http.Request, project string, segs []string) {
	if len(segs) == 0 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"operations": []any{}})
		return
	}
	writeError(w, gcperr.NotFound, "operation not found")
}

// --- helpers ---

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	raw, err := readBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "%v", err)
		return false
	}
	if err := json.Unmarshal(raw, v); err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON in request body: %v", err)
		return false
	}
	return true
}

func decodeBodyMap(w http.ResponseWriter, r *http.Request, v *map[string]any) bool {
	raw, err := readBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "%v", err)
		return false
	}
	if err := json.Unmarshal(raw, v); err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON in request body: %v", err)
		return false
	}
	return true
}

func readBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, errString("request body is required")
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		return nil, errString("failed to read request body")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errString("request body is required")
	}
	return raw, nil
}

type errString string

func (e errString) Error() string { return string(e) }

func parseMaxResults(w http.ResponseWriter, s string) (int, bool) {
	if s == "" {
		return defaultPageSize, true
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		writeError(w, gcperr.InvalidArgument, "invalid maxResults %q", s)
		return 0, false
	}
	if n == 0 {
		return defaultPageSize, true
	}
	return n, true
}

func toJSONBindings(in []iam.Binding) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, b := range in {
		out = append(out, map[string]any{"role": b.Role, "members": b.Members})
	}
	return out
}
