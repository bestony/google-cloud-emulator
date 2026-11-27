package gcs

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bestony/google-cloud-emulator/core/iam"
)

func newHandler() *Handler {
	return NewHandler(NewState(iam.NewStore()))
}

func do(h *Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
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

func TestObjectsUnimplemented(t *testing.T) {
	h := newHandler()
	do(h, "POST", "/storage/v1/b?project=demo", `{"name":"obj-bucket"}`)
	for _, path := range []string{
		"/storage/v1/b/obj-bucket/o",
		"/storage/v1/b/obj-bucket/o/some/object.txt",
		"/storage/v1/upload/storage/v1/b/obj-bucket/o",
	} {
		rec := do(h, "POST", path, `{}`)
		if rec.Code != 501 && rec.Code != 404 {
			t.Fatalf("%s: got %d, want 501", path, rec.Code)
		}
	}
}
