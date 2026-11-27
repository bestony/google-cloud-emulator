package gcs

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bestony/google-cloud-emulator/core/iam"
)

func newHandler() *Handler {
	return NewHandler(NewState(iam.NewStore()))
}

func do(h *Handler, method, path, body string) *httptest.ResponseRecorder {
	return doTyped(h, method, path, body, "")
}

func doTyped(h *Handler, method, path, body, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestBucketCRUD(t *testing.T) {
	h := newHandler()

	// List is empty initially.
	rec := do(h, "GET", "/storage/v1/b?project=demo", "")
	if rec.Code != 200 {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	var list map[string]any
	json.Unmarshal(rec.Body.Bytes(), &list)
	if items, _ := list["items"].([]any); len(items) != 0 {
		t.Fatalf("expected empty list, got %v", list["items"])
	}

	// Insert.
	rec = do(h, "POST", "/storage/v1/b?project=demo",
		`{"name":"my-bucket","location":"US-CENTRAL1","storageClass":"NEARLINE","labels":{"env":"test"}}`)
	if rec.Code != 200 {
		t.Fatalf("insert: %d %s", rec.Code, rec.Body)
	}
	var b map[string]any
	json.Unmarshal(rec.Body.Bytes(), &b)
	if b["kind"] != "storage#bucket" || b["name"] != "my-bucket" || b["project"] != "demo" {
		t.Fatalf("bad bucket: %v", b)
	}
	if b["metageneration"] != "1" || b["etag"] == "" || b["selfLink"] == "" {
		t.Fatalf("missing server fields: %v", b)
	}
	if b["storageClass"] != "NEARLINE" {
		t.Fatalf("client storageClass lost: %v", b["storageClass"])
	}

	// Duplicate insert -> 409.
	rec = do(h, "POST", "/storage/v1/b?project=demo", `{"name":"my-bucket"}`)
	if rec.Code != 409 {
		t.Fatalf("duplicate: got %d, want 409", rec.Code)
	}

	// Get.
	rec = do(h, "GET", "/storage/v1/b/my-bucket", "")
	if rec.Code != 200 {
		t.Fatalf("get: %d", rec.Code)
	}

	// Patch merges labels and bumps metageneration.
	rec = do(h, "PATCH", "/storage/v1/b/my-bucket", `{"labels":{"team":"platform"}}`)
	if rec.Code != 200 {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}
	json.Unmarshal(rec.Body.Bytes(), &b)
	if b["metageneration"] != "2" {
		t.Fatalf("metageneration not bumped: %v", b["metageneration"])
	}
	labels := b["labels"].(map[string]any)
	if labels["env"] != "test" || labels["team"] != "platform" {
		t.Fatalf("patch should merge labels: %v", labels)
	}

	// List now returns the bucket for project demo only.
	rec = do(h, "GET", "/storage/v1/b?project=demo", "")
	json.Unmarshal(rec.Body.Bytes(), &list)
	if items, _ := list["items"].([]any); len(items) != 1 {
		t.Fatalf("list: %v", list["items"])
	}
	rec = do(h, "GET", "/storage/v1/b?project=other", "")
	json.Unmarshal(rec.Body.Bytes(), &list)
	if items, _ := list["items"].([]any); len(items) != 0 {
		t.Fatalf("other project should be empty: %v", list["items"])
	}

	// Delete.
	rec = do(h, "DELETE", "/storage/v1/b/my-bucket", "")
	if rec.Code != 200 {
		t.Fatalf("delete: %d", rec.Code)
	}
	rec = do(h, "GET", "/storage/v1/b/my-bucket", "")
	if rec.Code != 404 {
		t.Fatalf("get after delete: got %d, want 404", rec.Code)
	}
}

func TestBucketValidation(t *testing.T) {
	h := newHandler()
	for _, name := range []string{"Bad_Name", "ab", "192.168.0.1", "a..b", "-lead", "trail-"} {
		rec := do(h, "POST", "/storage/v1/b?project=p", `{"name":"`+name+`"}`)
		if rec.Code != 400 {
			t.Fatalf("name %q: got %d, want 400 (%s)", name, rec.Code, rec.Body)
		}
	}
	// Missing project.
	if rec := do(h, "POST", "/storage/v1/b", `{"name":"ok-name"}`); rec.Code != 400 {
		t.Fatalf("missing project: got %d", rec.Code)
	}
	// List without project.
	if rec := do(h, "GET", "/storage/v1/b", ""); rec.Code != 400 {
		t.Fatalf("list without project: got %d", rec.Code)
	}
}

func TestBucketIAM(t *testing.T) {
	h := newHandler()
	do(h, "POST", "/storage/v1/b?project=demo", `{"name":"iam-bucket"}`)

	// Empty policy.
	rec := do(h, "GET", "/storage/v1/b/iam-bucket/iam", "")
	if rec.Code != 200 {
		t.Fatalf("get iam: %d", rec.Code)
	}
	var pol map[string]any
	json.Unmarshal(rec.Body.Bytes(), &pol)
	if pol["kind"] != "storage#policy" {
		t.Fatalf("bad policy: %v", pol)
	}

	// Set policy (GCS sends the Policy object directly).
	rec = do(h, "PUT", "/storage/v1/b/iam-bucket/iam",
		`{"version":1,"bindings":[{"role":"roles/storage.admin","members":["user:alice@example.com"]}]}`)
	if rec.Code != 200 {
		t.Fatalf("set iam: %d %s", rec.Code, rec.Body)
	}
	json.Unmarshal(rec.Body.Bytes(), &pol)
	bindings := pol["bindings"].([]any)
	if len(bindings) != 1 {
		t.Fatalf("bindings: %v", bindings)
	}

	// Test permissions: caller is dev@localhost (allUsers/allAuthenticatedUsers).
	rec = do(h, "POST", "/storage/v1/b/iam-bucket/iam/test", `{"permissions":["storage.buckets.get","storage.buckets.setIamPolicy"]}`)
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	perms := resp["permissions"].([]any)
	// No binding matches the caller -> empty.
	if len(perms) != 0 {
		t.Fatalf("expected no permissions, got %v", perms)
	}

	// Bucket IAM on missing bucket -> 404.
	rec = do(h, "GET", "/storage/v1/b/nope/iam", "")
	if rec.Code != 404 {
		t.Fatalf("iam on missing bucket: got %d", rec.Code)
	}
}

func TestObjectDataPlane(t *testing.T) {
	h := newHandler()
	do(h, "POST", "/storage/v1/b?project=demo", `{"name":"obj-bucket"}`)

	// Simple media upload creates object metadata with hashes and server fields.
	rec := doTyped(h, "POST", "/upload/storage/v1/b/obj-bucket/o?uploadType=media&name=dir/hello.txt", "hello world", "text/plain")
	if rec.Code != 200 {
		t.Fatalf("media upload: %d %s", rec.Code, rec.Body)
	}
	var obj map[string]any
	json.Unmarshal(rec.Body.Bytes(), &obj)
	if obj["kind"] != "storage#object" || obj["bucket"] != "obj-bucket" || obj["name"] != "dir/hello.txt" {
		t.Fatalf("bad object metadata: %v", obj)
	}
	if obj["size"] != "11" || obj["md5Hash"] == "" || obj["crc32c"] == "" || obj["generation"] == "" {
		t.Fatalf("missing object data fields: %v", obj)
	}
	if obj["contentType"] != "text/plain" {
		t.Fatalf("content type not preserved: %v", obj["contentType"])
	}
	generation := obj["generation"].(string)

	// Metadata is addressable using the URL-escaped object name.
	rec = do(h, "GET", "/storage/v1/b/obj-bucket/o/dir%2Fhello.txt", "")
	if rec.Code != 200 {
		t.Fatalf("get object metadata: %d %s", rec.Code, rec.Body)
	}

	// alt=media downloads the bytes and content type.
	rec = do(h, "GET", "/storage/v1/b/obj-bucket/o/dir%2Fhello.txt?alt=media", "")
	if rec.Code != 200 || rec.Body.String() != "hello world" {
		t.Fatalf("download: %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("download content type: %q", ct)
	}

	// Generation-specific reads work.
	rec = do(h, "GET", "/storage/v1/b/obj-bucket/o/dir%2Fhello.txt?alt=media&generation="+generation, "")
	if rec.Code != 200 || rec.Body.String() != "hello world" {
		t.Fatalf("generation download: %d %q", rec.Code, rec.Body.String())
	}

	// PATCH merges custom metadata and bumps metageneration.
	rec = do(h, "PATCH", "/storage/v1/b/obj-bucket/o/dir%2Fhello.txt", `{"metadata":{"tier":"hot"},"cacheControl":"no-cache"}`)
	if rec.Code != 200 {
		t.Fatalf("patch object: %d %s", rec.Code, rec.Body)
	}
	json.Unmarshal(rec.Body.Bytes(), &obj)
	if obj["metageneration"] != "2" || obj["cacheControl"] != "no-cache" {
		t.Fatalf("patch did not update metadata: %v", obj)
	}
	custom := obj["metadata"].(map[string]any)
	if custom["tier"] != "hot" {
		t.Fatalf("custom metadata lost: %v", custom)
	}

	// Objects can be listed by prefix.
	rec = do(h, "GET", "/storage/v1/b/obj-bucket/o?prefix=dir/", "")
	if rec.Code != 200 {
		t.Fatalf("list objects: %d", rec.Code)
	}
	var list map[string]any
	json.Unmarshal(rec.Body.Bytes(), &list)
	if items, _ := list["items"].([]any); len(items) != 1 {
		t.Fatalf("list objects: %v", list)
	}

	// Delete removes the live object.
	rec = do(h, "DELETE", "/storage/v1/b/obj-bucket/o/dir%2Fhello.txt", "")
	if rec.Code != 200 {
		t.Fatalf("delete object: %d %s", rec.Code, rec.Body)
	}
	rec = do(h, "GET", "/storage/v1/b/obj-bucket/o/dir%2Fhello.txt", "")
	if rec.Code != 404 {
		t.Fatalf("get after delete: got %d, want 404", rec.Code)
	}
}

func TestObjectUploadsCopyComposeAndListDelimiter(t *testing.T) {
	h := newHandler()
	do(h, "POST", "/storage/v1/b?project=demo", `{"name":"data-bucket"}`)

	// Multipart upload carries JSON metadata plus object bytes in one request.
	boundary := "gcptest"
	body := "--" + boundary + "\r\nContent-Type: application/json; charset=UTF-8\r\n\r\n" +
		`{"name":"dir/a.txt","metadata":{"part":"one"}}` + "\r\n" +
		"--" + boundary + "\r\nContent-Type: text/plain\r\n\r\nA\r\n" +
		"--" + boundary + "--\r\n"
	rec := doTyped(h, "POST", "/upload/storage/v1/b/data-bucket/o?uploadType=multipart", body, "multipart/related; boundary="+boundary)
	if rec.Code != 200 {
		t.Fatalf("multipart upload: %d %s", rec.Code, rec.Body)
	}
	var obj map[string]any
	json.Unmarshal(rec.Body.Bytes(), &obj)
	if obj["name"] != "dir/a.txt" || obj["size"] != "1" {
		t.Fatalf("multipart metadata: %v", obj)
	}

	// A second object exercises delimiter listing and compose.
	doTyped(h, "POST", "/upload/storage/v1/b/data-bucket/o?uploadType=media&name=dir/sub/b.txt", "B", "text/plain")
	doTyped(h, "POST", "/upload/storage/v1/b/data-bucket/o?uploadType=media&name=root.txt", "R", "text/plain")
	rec = do(h, "GET", "/storage/v1/b/data-bucket/o?prefix=dir/&delimiter=/", "")
	if rec.Code != 200 {
		t.Fatalf("delimiter list: %d", rec.Code)
	}
	var list map[string]any
	json.Unmarshal(rec.Body.Bytes(), &list)
	if items, _ := list["items"].([]any); len(items) != 1 {
		t.Fatalf("delimiter list items: %v", list)
	}
	if prefixes, _ := list["prefixes"].([]any); len(prefixes) != 1 || prefixes[0] != "dir/sub/" {
		t.Fatalf("delimiter list prefixes: %v", list)
	}

	// copyTo duplicates bytes and metadata under a new name.
	rec = do(h, "POST", "/storage/v1/b/data-bucket/o/dir%2Fa.txt/copyTo/b/data-bucket/o/copied.txt", `{}`)
	if rec.Code != 200 {
		t.Fatalf("copyTo: %d %s", rec.Code, rec.Body)
	}
	rec = do(h, "GET", "/storage/v1/b/data-bucket/o/copied.txt?alt=media", "")
	if rec.Code != 200 || rec.Body.String() != "A" {
		t.Fatalf("copied download: %d %q", rec.Code, rec.Body.String())
	}

	// compose concatenates source object bytes in request order.
	rec = do(h, "POST", "/storage/v1/b/data-bucket/o/joined.txt/compose", `{"sourceObjects":[{"name":"dir/a.txt"},{"name":"dir/sub/b.txt"}]}`)
	if rec.Code != 200 {
		t.Fatalf("compose: %d %s", rec.Code, rec.Body)
	}
	rec = do(h, "GET", "/storage/v1/b/data-bucket/o/joined.txt?alt=media", "")
	if rec.Code != 200 || rec.Body.String() != "AB" {
		t.Fatalf("composed download: %d %q", rec.Code, rec.Body.String())
	}
}

func TestObjectResumableUploadAndVersioning(t *testing.T) {
	h := newHandler()
	do(h, "POST", "/storage/v1/b?project=demo", `{"name":"versioned-bucket","versioning":{"enabled":true}}`)

	// Initiate a resumable upload session.
	rec := doTyped(h, "POST", "/upload/storage/v1/b/versioned-bucket/o?uploadType=resumable", `{"name":"chunked.bin","contentType":"application/octet-stream"}`, "application/json")
	if rec.Code != 200 {
		t.Fatalf("start resumable: %d %s", rec.Code, rec.Body)
	}
	loc := rec.Header().Get("Location")
	if loc == "" {
		t.Fatalf("missing resumable session Location")
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}

	// A single PUT completes the session.
	rec = doTyped(h, "PUT", u.Path+"?"+u.RawQuery, "chunked-content", "application/octet-stream")
	if rec.Code != 200 {
		t.Fatalf("complete resumable: %d %s", rec.Code, rec.Body)
	}
	var first map[string]any
	json.Unmarshal(rec.Body.Bytes(), &first)
	firstGen := first["generation"].(string)

	// Overwriting keeps the old generation because bucket versioning is enabled.
	rec = doTyped(h, "POST", "/upload/storage/v1/b/versioned-bucket/o?uploadType=media&name=chunked.bin", "new-content", "application/octet-stream")
	if rec.Code != 200 {
		t.Fatalf("overwrite: %d %s", rec.Code, rec.Body)
	}
	var second map[string]any
	json.Unmarshal(rec.Body.Bytes(), &second)
	if second["generation"] == firstGen {
		t.Fatalf("overwrite did not allocate a new generation")
	}
	rec = do(h, "GET", "/storage/v1/b/versioned-bucket/o/chunked.bin?versions=true", "")
	// Metadata GET ignores versions=true; list is the versioned view.
	if rec.Code != 200 {
		t.Fatalf("metadata after overwrite: %d", rec.Code)
	}
	rec = do(h, "GET", "/storage/v1/b/versioned-bucket/o?versions=true&prefix=chunked.bin", "")
	var list map[string]any
	json.Unmarshal(rec.Body.Bytes(), &list)
	if items, _ := list["items"].([]any); len(items) != 2 {
		t.Fatalf("versioned list should contain live and archived generation: %v", list)
	}

	// The archived generation remains readable by generation.
	rec = do(h, "GET", "/storage/v1/b/versioned-bucket/o/chunked.bin?alt=media&generation="+firstGen, "")
	if rec.Code != 200 || rec.Body.String() != "chunked-content" {
		t.Fatalf("archived generation download: %d %q", rec.Code, rec.Body.String())
	}
}
