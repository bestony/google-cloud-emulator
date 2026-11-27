package gcs

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

var objectServerManagedKeys = map[string]bool{
	"kind": true, "id": true, "selfLink": true, "bucket": true, "name": true,
	"generation": true, "metageneration": true, "size": true, "md5Hash": true,
	"crc32c": true, "etag": true, "timeCreated": true, "updated": true,
	"componentCount": true, "timeStorageClassUpdated": true, "timeFinalized": true,
}

func objectSelfLink(bucket, name string) string {
	return "https://storage.googleapis.com/storage/v1/b/" + url.PathEscape(bucket) + "/o/" + url.PathEscape(name)
}

func metaInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	default:
		return 0
	}
}

func objectContentType(meta map[string]any) string {
	if ct, ok := meta["contentType"].(string); ok && ct != "" {
		return ct
	}
	return "application/octet-stream"
}

func objectChecksums(data []byte) (md5Hash, crc32c string) {
	sum := md5.Sum(data)
	md5Hash = base64.StdEncoding.EncodeToString(sum[:])
	crc := crc32.Checksum(data, crc32cTable)
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], crc)
	crc32c = base64.StdEncoding.EncodeToString(raw[:])
	return md5Hash, crc32c
}

func cleanObjectClientMetadata(input map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range input {
		if !objectServerManagedKeys[k] {
			out[k] = cloneValue(v)
		}
	}
	return out
}

func (h *Handler) buildObjectMetadata(bucket, name string, data []byte, client map[string]any, generation, metageneration int64, timeCreated time.Time, componentCount int) map[string]any {
	stored := cleanObjectClientMetadata(client)
	now := h.now()
	if stored["contentType"] == nil || stored["contentType"] == "" {
		stored["contentType"] = "application/octet-stream"
	}
	if stored["storageClass"] == nil || stored["storageClass"] == "" {
		stored["storageClass"] = "STANDARD"
	}
	md5Hash, crc32c := objectChecksums(data)
	stored["kind"] = "storage#object"
	stored["id"] = fmt.Sprintf("%s/%s/%d", bucket, name, generation)
	stored["selfLink"] = objectSelfLink(bucket, name)
	stored["bucket"] = bucket
	stored["name"] = name
	stored["generation"] = strconv.FormatInt(generation, 10)
	stored["metageneration"] = strconv.FormatInt(metageneration, 10)
	stored["size"] = strconv.Itoa(len(data))
	stored["md5Hash"] = md5Hash
	stored["crc32c"] = crc32c
	stored["timeCreated"] = timeCreated.Format(time.RFC3339)
	stored["updated"] = now.Format(time.RFC3339)
	stored["timeStorageClassUpdated"] = stored["timeCreated"]
	if componentCount > 0 {
		stored["componentCount"] = strconv.Itoa(componentCount)
	}
	delete(stored, "etag")
	stored["etag"] = bucketETag(stored)
	return stored
}

func refreshObjectMetadata(meta map[string]any, dataLen int, updated time.Time) {
	md5Hash, crc32c := objectChecksums([]byte{})
	_ = md5Hash
	_ = crc32c
	meta["size"] = strconv.Itoa(dataLen)
	meta["updated"] = updated.Format(time.RFC3339)
	delete(meta, "etag")
	meta["etag"] = bucketETag(meta)
}

func parseGeneration(raw string) (int64, bool) {
	if raw == "" {
		return 0, true
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func validObjectName(name string) bool {
	if name == "" || len(name) > 1024 {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func (h *Handler) routeObjects(w http.ResponseWriter, r *http.Request, bucket string, segs []string) {
	if len(segs) == 0 {
		if r.Method != http.MethodGet {
			writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
			return
		}
		h.objectsList(w, r, bucket)
		return
	}

	// copyTo and rewriteTo encode the source object before the literal
	// operation segments and the destination object after /b/{bucket}/o/.
	for i, seg := range segs {
		if (seg == "copyTo" || seg == "rewriteTo") && i+3 < len(segs) && segs[i+1] == "b" && segs[i+3] == "o" {
			if r.Method != http.MethodPost {
				writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
				return
			}
			source := strings.Join(segs[:i], "/")
			destBucket := segs[i+2]
			destName := strings.Join(segs[i+4:], "/")
			h.objectCopy(w, r, bucket, source, destBucket, destName, seg == "rewriteTo")
			return
		}
	}
	if segs[len(segs)-1] == "compose" && len(segs) > 1 {
		if r.Method != http.MethodPost {
			writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
			return
		}
		h.objectCompose(w, r, bucket, strings.Join(segs[:len(segs)-1], "/"))
		return
	}

	name := strings.Join(segs, "/")
	if !validObjectName(name) {
		writeError(w, gcperr.InvalidArgument, "invalid object name %q", name)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.objectGet(w, r, bucket, name)
	case http.MethodPatch:
		h.objectPatch(w, r, bucket, name)
	case http.MethodPut:
		h.objectUpdate(w, r, bucket, name)
	case http.MethodDelete:
		h.objectDelete(w, r, bucket, name)
	default:
		writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
	}
}

type objectListRow struct {
	key      string
	metadata map[string]any
	prefix   bool
}

func (h *Handler) objectsList(w http.ResponseWriter, r *http.Request, bucket string) {
	if _, ok := h.state.get(bucket); !ok {
		writeError(w, gcperr.NotFound, "bucket %q not found", bucket)
		return
	}
	q := r.URL.Query()
	prefix := q.Get("prefix")
	delimiter := q.Get("delimiter")
	startOffset := q.Get("startOffset")
	endOffset := q.Get("endOffset")
	includeVersions := q.Get("versions") == "true"
	includeTrailingDelimiter := q.Get("includeTrailingDelimiter") == "true"
	maxResults := 1000
	if raw := q.Get("maxResults"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > 1000 {
			writeError(w, gcperr.InvalidArgument, "invalid maxResults %q", raw)
			return
		}
		maxResults = n
	}
	start := 0
	if token := q.Get("pageToken"); token != "" {
		n, err := strconv.Atoi(token)
		if err != nil || n < 0 {
			writeError(w, gcperr.InvalidArgument, "invalid pageToken %q", token)
			return
		}
		start = n
	}

	entries := h.state.listObjects(bucket, includeVersions)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name == entries[j].Name {
			return metaInt64(entries[i].Metadata["generation"]) > metaInt64(entries[j].Metadata["generation"])
		}
		return entries[i].Name < entries[j].Name
	})

	prefixes := map[string]bool{}
	rows := []objectListRow{}
	for _, entry := range entries {
		name := entry.Name
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if startOffset != "" && name < startOffset {
			continue
		}
		if endOffset != "" && name >= endOffset {
			continue
		}
		if delimiter != "" {
			rest := strings.TrimPrefix(name, prefix)
			if idx := strings.Index(rest, delimiter); idx >= 0 {
				prefixKey := prefix + rest[:idx+len(delimiter)]
				if name == prefixKey && includeTrailingDelimiter {
					rows = append(rows, objectListRow{key: name, metadata: entry.Metadata})
				} else {
					prefixes[prefixKey] = true
				}
				continue
			}
		}
		rows = append(rows, objectListRow{key: name, metadata: entry.Metadata})
	}
	for p := range prefixes {
		rows = append(rows, objectListRow{key: p, prefix: true})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].key < rows[j].key })

	if start > len(rows) {
		start = len(rows)
	}
	end := start + maxResults
	if end > len(rows) {
		end = len(rows)
	}
	items := []map[string]any{}
	outPrefixes := []string{}
	for _, row := range rows[start:end] {
		if row.prefix {
			outPrefixes = append(outPrefixes, row.key)
		} else {
			items = append(items, row.metadata)
		}
	}
	resp := map[string]any{"kind": "storage#objects"}
	if len(items) > 0 {
		resp["items"] = items
	}
	if len(outPrefixes) > 0 {
		resp["prefixes"] = outPrefixes
	}
	if end < len(rows) {
		resp["nextPageToken"] = strconv.Itoa(end)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) objectGet(w http.ResponseWriter, r *http.Request, bucket, name string) {
	generation, ok := parseGeneration(r.URL.Query().Get("generation"))
	if !ok {
		writeError(w, gcperr.InvalidArgument, "invalid generation")
		return
	}
	meta, data, found := h.state.getObject(bucket, name, generation)
	if !found {
		writeError(w, gcperr.NotFound, "object %q not found", name)
		return
	}
	if r.URL.Query().Get("alt") == "media" {
		w.Header().Set("Content-Type", objectContentType(meta))
		w.Header().Set("ETag", fmt.Sprint(meta["etag"]))
		w.Header().Set("X-Goog-Generation", fmt.Sprint(meta["generation"]))
		w.Header().Set("X-Goog-Metageneration", fmt.Sprint(meta["metageneration"]))
		http.ServeContent(w, r, name, h.now(), bytes.NewReader(data))
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (h *Handler) objectPatch(w http.ResponseWriter, r *http.Request, bucket, name string) {
	generation, ok := parseGeneration(r.URL.Query().Get("generation"))
	if !ok {
		writeError(w, gcperr.InvalidArgument, "invalid generation")
		return
	}
	meta, data, found := h.state.getObject(bucket, name, generation)
	if !found {
		writeError(w, gcperr.NotFound, "object %q not found", name)
		return
	}
	patch, err := h.readBucketBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON body: %v", err)
		return
	}
	for k := range patch {
		if objectServerManagedKeys[k] {
			delete(patch, k)
		}
	}
	mergePatch(meta, patch)
	meta["metageneration"] = strconv.FormatInt(metaInt64(meta["metageneration"])+1, 10)
	refreshObjectMetadata(meta, len(data), h.now())
	if !h.state.replaceObjectMetadata(bucket, name, metaInt64(meta["generation"]), meta) {
		writeError(w, gcperr.NotFound, "object %q not found", name)
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (h *Handler) objectUpdate(w http.ResponseWriter, r *http.Request, bucket, name string) {
	generation, ok := parseGeneration(r.URL.Query().Get("generation"))
	if !ok {
		writeError(w, gcperr.InvalidArgument, "invalid generation")
		return
	}
	old, data, found := h.state.getObject(bucket, name, generation)
	if !found {
		writeError(w, gcperr.NotFound, "object %q not found", name)
		return
	}
	body, err := h.readBucketBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON body: %v", err)
		return
	}
	meta := cleanObjectClientMetadata(body)
	for _, key := range []string{"kind", "id", "selfLink", "bucket", "name", "generation", "timeCreated", "timeStorageClassUpdated"} {
		meta[key] = old[key]
	}
	meta["metageneration"] = strconv.FormatInt(metaInt64(old["metageneration"])+1, 10)
	md5Hash, crc32c := objectChecksums(data)
	meta["md5Hash"] = md5Hash
	meta["crc32c"] = crc32c
	refreshObjectMetadata(meta, len(data), h.now())
	if !h.state.replaceObjectMetadata(bucket, name, metaInt64(old["generation"]), meta) {
		writeError(w, gcperr.NotFound, "object %q not found", name)
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (h *Handler) objectDelete(w http.ResponseWriter, r *http.Request, bucket, name string) {
	generation, ok := parseGeneration(r.URL.Query().Get("generation"))
	if !ok {
		writeError(w, gcperr.InvalidArgument, "invalid generation")
		return
	}
	if !h.state.deleteObject(bucket, name, generation) {
		writeError(w, gcperr.NotFound, "object %q not found", name)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *Handler) insertObject(w http.ResponseWriter, bucket, name string, data []byte, client map[string]any, contentType string) {
	if !validObjectName(name) {
		writeError(w, gcperr.InvalidArgument, "invalid object name %q", name)
		return
	}
	if _, ok := h.state.get(bucket); !ok {
		writeError(w, gcperr.NotFound, "bucket %q not found", bucket)
		return
	}
	if client == nil {
		client = map[string]any{}
	}
	if contentType != "" {
		if _, exists := client["contentType"]; !exists {
			client["contentType"] = contentType
		}
	}
	generation := h.state.newGeneration()
	meta := h.buildObjectMetadata(bucket, name, data, client, generation, 1, h.now(), 0)
	if !h.state.putObjectVersion(bucket, name, meta, data, generation) {
		writeError(w, gcperr.NotFound, "bucket %q not found", bucket)
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (h *Handler) routeUpload(w http.ResponseWriter, r *http.Request, segs []string) {
	if len(segs) < 3 || segs[0] != "b" || segs[2] != "o" {
		writeError(w, gcperr.NotFound, "unknown upload path %q", r.URL.Path)
		return
	}
	bucket := segs[1]
	q := r.URL.Query()
	if uploadID := q.Get("upload_id"); uploadID != "" {
		if r.Method != http.MethodPut && r.Method != http.MethodPost {
			writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
			return
		}
		h.resumableUploadChunk(w, r, uploadID)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, gcperr.InvalidArgument, "unsupported method %s", r.Method)
		return
	}
	switch q.Get("uploadType") {
	case "media":
		data, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, gcperr.InvalidArgument, "failed to read upload body: %v", err)
			return
		}
		h.insertObject(w, bucket, q.Get("name"), data, map[string]any{}, r.Header.Get("Content-Type"))
	case "multipart":
		h.multipartUpload(w, r, bucket)
	case "resumable":
		h.startResumableUpload(w, r, bucket)
	default:
		writeError(w, gcperr.InvalidArgument, "uploadType must be media, multipart, or resumable")
	}
}

func (h *Handler) multipartUpload(w http.ResponseWriter, r *http.Request, bucket string) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") || params["boundary"] == "" {
		writeError(w, gcperr.InvalidArgument, "multipart upload requires a multipart content type with boundary")
		return
	}
	reader := multipart.NewReader(r.Body, params["boundary"])
	var metadata map[string]any
	var data []byte
	contentType := ""
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeError(w, gcperr.InvalidArgument, "invalid multipart body: %v", err)
			return
		}
		raw, err := io.ReadAll(part)
		if err != nil {
			writeError(w, gcperr.InvalidArgument, "invalid multipart part: %v", err)
			return
		}
		ct := part.Header.Get("Content-Type")
		if strings.Contains(ct, "json") && metadata == nil {
			if err := json.Unmarshal(raw, &metadata); err != nil {
				writeError(w, gcperr.InvalidArgument, "invalid multipart metadata JSON: %v", err)
				return
			}
			continue
		}
		data = append(data, raw...)
		if contentType == "" {
			contentType = ct
		}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	name, _ := metadata["name"].(string)
	if qName := r.URL.Query().Get("name"); qName != "" {
		name = qName
	}
	h.insertObject(w, bucket, name, data, metadata, contentType)
}

func (h *Handler) startResumableUpload(w http.ResponseWriter, r *http.Request, bucket string) {
	metadata := map[string]any{}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "failed to read metadata body: %v", err)
		return
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &metadata); err != nil {
			writeError(w, gcperr.InvalidArgument, "invalid resumable metadata JSON: %v", err)
			return
		}
	}
	if qName := r.URL.Query().Get("name"); qName != "" {
		metadata["name"] = qName
	}
	name, _ := metadata["name"].(string)
	if !validObjectName(name) {
		writeError(w, gcperr.InvalidArgument, "invalid object name %q", name)
		return
	}
	if ct := r.Header.Get("X-Upload-Content-Type"); ct != "" {
		if _, exists := metadata["contentType"]; !exists {
			metadata["contentType"] = ct
		}
	}
	id, ok := h.state.createUploadSession(bucket, metadata)
	if !ok {
		writeError(w, gcperr.NotFound, "bucket %q not found", bucket)
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	location := scheme + "://" + r.Host + "/upload/storage/v1/b/" + url.PathEscape(bucket) + "/o?uploadType=resumable&upload_id=" + url.QueryEscape(id)
	w.Header().Set("Location", location)
	w.Header().Set("X-Upload-Upload-Type", "resumable")
	w.WriteHeader(http.StatusOK)
}

func parseContentRange(raw string) (start, end, total int64, ok bool) {
	if raw == "" || !strings.HasPrefix(raw, "bytes ") {
		return 0, 0, 0, false
	}
	spec := strings.TrimPrefix(raw, "bytes ")
	parts := strings.Split(spec, "/")
	if len(parts) != 2 {
		return 0, 0, 0, false
	}
	total, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || total < 0 {
		return 0, 0, 0, false
	}
	rangeParts := strings.Split(parts[0], "-")
	if len(rangeParts) != 2 {
		return 0, 0, 0, false
	}
	start, err1 := strconv.ParseInt(rangeParts[0], 10, 64)
	end, err2 := strconv.ParseInt(rangeParts[1], 10, 64)
	if err1 != nil || err2 != nil || start < 0 || end < start {
		return 0, 0, 0, false
	}
	return start, end, total, true
}

func (h *Handler) resumableUploadChunk(w http.ResponseWriter, r *http.Request, uploadID string) {
	bucket, metadata, existing, ok := h.state.uploadSessionByID(uploadID)
	if !ok {
		writeError(w, gcperr.NotFound, "upload session %q not found", uploadID)
		return
	}
	chunk, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "failed to read upload chunk: %v", err)
		return
	}
	start, end, total, ranged := parseContentRange(r.Header.Get("Content-Range"))
	if ranged {
		if start != int64(len(existing)) || end-start+1 != int64(len(chunk)) || end >= total {
			writeError(w, gcperr.InvalidArgument, "invalid Content-Range for upload chunk")
			return
		}
	}
	if !h.state.appendUploadSessionData(uploadID, chunk) {
		writeError(w, gcperr.NotFound, "upload session %q not found", uploadID)
		return
	}
	data := append(existing, chunk...)
	if ranged && end+1 < total {
		w.Header().Set("Range", fmt.Sprintf("bytes=0-%d", end))
		w.WriteHeader(308)
		return
	}
	name, _ := metadata["name"].(string)
	contentType, _ := metadata["contentType"].(string)
	h.state.deleteUploadSession(uploadID)
	h.insertObject(w, bucket, name, data, metadata, contentType)
}

func (h *Handler) objectCopy(w http.ResponseWriter, r *http.Request, sourceBucket, sourceName, destBucket, destName string, rewrite bool) {
	if !validObjectName(sourceName) || !validObjectName(destName) {
		writeError(w, gcperr.InvalidArgument, "invalid source or destination object name")
		return
	}
	generation, ok := parseGeneration(r.URL.Query().Get("sourceGeneration"))
	if !ok {
		writeError(w, gcperr.InvalidArgument, "invalid sourceGeneration")
		return
	}
	sourceMeta, data, found := h.state.getObject(sourceBucket, sourceName, generation)
	if !found {
		writeError(w, gcperr.NotFound, "source object %q not found", sourceName)
		return
	}
	body, err := h.readBucketBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON body: %v", err)
		return
	}
	client := cleanObjectClientMetadata(sourceMeta)
	for k, v := range cleanObjectClientMetadata(body) {
		client[k] = v
	}
	gen := h.state.newGeneration()
	meta := h.buildObjectMetadata(destBucket, destName, data, client, gen, 1, h.now(), 0)
	if !h.state.putObjectVersion(destBucket, destName, meta, data, gen) {
		writeError(w, gcperr.NotFound, "destination bucket %q not found", destBucket)
		return
	}
	if !rewrite {
		writeJSON(w, http.StatusOK, meta)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind":     "storage#rewriteResponse",
		"done":     true,
		"resource": meta,
	})
}

func (h *Handler) objectCompose(w http.ResponseWriter, r *http.Request, bucket, destName string) {
	if !validObjectName(destName) {
		writeError(w, gcperr.InvalidArgument, "invalid object name %q", destName)
		return
	}
	body, err := h.readBucketBody(r)
	if err != nil {
		writeError(w, gcperr.InvalidArgument, "invalid JSON body: %v", err)
		return
	}
	rawSources, _ := body["sourceObjects"].([]any)
	if len(rawSources) == 0 {
		writeError(w, gcperr.InvalidArgument, "sourceObjects is required")
		return
	}
	if len(rawSources) > 32 {
		writeError(w, gcperr.InvalidArgument, "at most 32 source objects can be composed")
		return
	}
	data := []byte{}
	firstContentType := ""
	for _, raw := range rawSources {
		src, _ := raw.(map[string]any)
		if src == nil {
			writeError(w, gcperr.InvalidArgument, "sourceObjects entries must be objects")
			return
		}
		name, _ := src["name"].(string)
		if !validObjectName(name) {
			writeError(w, gcperr.InvalidArgument, "invalid source object name %q", name)
			return
		}
		generation := metaInt64(src["generation"])
		meta, bytes, found := h.state.getObject(bucket, name, generation)
		if !found {
			writeError(w, gcperr.NotFound, "source object %q not found", name)
			return
		}
		if firstContentType == "" {
			firstContentType = objectContentType(meta)
		}
		data = append(data, bytes...)
	}
	client := map[string]any{}
	if destination, _ := body["destination"].(map[string]any); destination != nil {
		client = cleanObjectClientMetadata(destination)
	} else {
		client = cleanObjectClientMetadata(body)
	}
	if _, exists := client["contentType"]; !exists {
		client["contentType"] = firstContentType
	}
	gen := h.state.newGeneration()
	meta := h.buildObjectMetadata(bucket, destName, data, client, gen, 1, h.now(), len(rawSources))
	if !h.state.putObjectVersion(bucket, destName, meta, data, gen) {
		writeError(w, gcperr.NotFound, "bucket %q not found", bucket)
		return
	}
	writeJSON(w, http.StatusOK, meta)
}
