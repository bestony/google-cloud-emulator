package dns

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/bestony/google-cloud-emulator/core/iam"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	return NewHandler(NewState(iam.NewStore()))
}

func do(t *testing.T, h *Handler, method, path, body string) (int, map[string]any) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	raw, _ := io.ReadAll(rec.Result().Body)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return rec.Code, out
}

func TestZoneLifecycle(t *testing.T) {
	h := newTestHandler(t)

	// projects.get
	code, proj := do(t, h, "GET", "/dns/v1/projects/test-project", "")
	if code != 200 || proj["kind"] != "dns#project" {
		t.Fatalf("projects.get: %d %v", code, proj)
	}

	// create zone
	code, zone := do(t, h, "POST", "/dns/v1/projects/test-project/managedZones",
		`{"name":"example-com","dnsName":"example.com.","description":"test zone"}`)
	if code != 200 {
		t.Fatalf("create zone: %d %v", code, zone)
	}
	if zone["kind"] != "dns#managedZone" || zone["name"] != "example-com" {
		t.Fatalf("unexpected zone: %v", zone)
	}
	if ns, ok := zone["nameServers"].([]any); !ok || len(ns) != 4 {
		t.Fatalf("expected 4 name servers, got %v", zone["nameServers"])
	}

	// duplicate create -> 409
	code, _ = do(t, h, "POST", "/dns/v1/projects/test-project/managedZones",
		`{"name":"example-com","dnsName":"example.com."}`)
	if code != 409 {
		t.Fatalf("duplicate zone: got %d", code)
	}

	// get
	code, got := do(t, h, "GET", "/dns/v1/projects/test-project/managedZones/example-com", "")
	if code != 200 || got["dnsName"] != "example.com." {
		t.Fatalf("get zone: %d %v", code, got)
	}

	// patch description
	code, patched := do(t, h, "PATCH", "/dns/v1/projects/test-project/managedZones/example-com",
		`{"description":"updated"}`)
	if code != 200 || patched["description"] != "updated" {
		t.Fatalf("patch zone: %d %v", code, patched)
	}

	// list
	code, list := do(t, h, "GET", "/dns/v1/projects/test-project/managedZones", "")
	if code != 200 || list["kind"] != "dns#managedZonesListResponse" {
		t.Fatalf("list zones: %d %v", code, list)
	}
	if zones, _ := list["managedZones"].([]any); len(zones) != 1 {
		t.Fatalf("expected 1 zone, got %v", list["managedZones"])
	}

	// changes: add an A record
	code, chg := do(t, h, "POST", "/dns/v1/projects/test-project/managedZones/example-com/changes",
		`{"additions":[{"name":"www.example.com.","type":"A","ttl":300,"rrdatas":["192.0.2.1"]}]}`)
	if code != 200 {
		t.Fatalf("create change: %d %v", code, chg)
	}
	if chg["status"] != "done" || chg["kind"] != "dns#change" {
		t.Fatalf("unexpected change: %v", chg)
	}

	// rrsets list should include apex NS + new A record
	code, rrsets := do(t, h, "GET", "/dns/v1/projects/test-project/managedZones/example-com/rrsets", "")
	if code != 200 {
		t.Fatalf("list rrsets: %d %v", code, rrsets)
	}
	rs, _ := rrsets["rrsets"].([]any)
	if len(rs) != 2 {
		t.Fatalf("expected 2 rrsets, got %v", rs)
	}

	// rrsets get
	code, one := do(t, h, "GET", "/dns/v1/projects/test-project/managedZones/example-com/rrsets/www.example.com./A", "")
	if code != 200 {
		t.Fatalf("get rrset: %d %v", code, one)
	}

	// changes list/get
	code, clist := do(t, h, "GET", "/dns/v1/projects/test-project/managedZones/example-com/changes", "")
	if code != 200 {
		t.Fatalf("list changes: %d %v", code, clist)
	}
	changes, _ := clist["changes"].([]any)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %v", changes)
	}
	code, cget := do(t, h, "GET", "/dns/v1/projects/test-project/managedZones/example-com/changes/1", "")
	if code != 200 || cget["id"] != "1" {
		t.Fatalf("get change: %d %v", code, cget)
	}

	// delete the A record via change, then zone becomes deletable
	code, _ = do(t, h, "POST", "/dns/v1/projects/test-project/managedZones/example-com/changes",
		`{"deletions":[{"name":"www.example.com.","type":"A","ttl":300,"rrdatas":["192.0.2.1"]}]}`)
	if code != 200 {
		t.Fatalf("delete rrset change: %d", code)
	}
	// zone with remaining records must refuse deletion
	code, _ = do(t, h, "POST", "/dns/v1/projects/test-project/managedZones/example-com/changes",
		`{"additions":[{"name":"mail.example.com.","type":"MX","ttl":300,"rrdatas":["10 mail.example.com."]}]}`)
	if code != 200 {
		t.Fatalf("add mx: %d", code)
	}
	code, _ = do(t, h, "DELETE", "/dns/v1/projects/test-project/managedZones/example-com", "")
	if code != 400 {
		t.Fatalf("delete non-empty zone: got %d, want 400", code)
	}
	code, _ = do(t, h, "POST", "/dns/v1/projects/test-project/managedZones/example-com/changes",
		`{"deletions":[{"name":"mail.example.com.","type":"MX","ttl":300,"rrdatas":["10 mail.example.com."]}]}`)
	if code != 200 {
		t.Fatalf("delete mx change: %d", code)
	}
	code, _ = do(t, h, "DELETE", "/dns/v1/projects/test-project/managedZones/example-com", "")
	if code != 200 {
		t.Fatalf("delete zone: got %d", code)
	}
	code, _ = do(t, h, "GET", "/dns/v1/projects/test-project/managedZones/example-com", "")
	if code != 404 {
		t.Fatalf("get deleted zone: got %d, want 404", code)
	}
}

func TestDNSIAM(t *testing.T) {
	h := newTestHandler(t)
	code, _ := do(t, h, "POST", "/dns/v1/projects/p/managedZones", `{"name":"z1","dnsName":"z1.example."}`)
	if code != 200 {
		t.Fatalf("create zone: %d", code)
	}

	// set policy
	code, pol := do(t, h, "POST", "/dns/v1/projects/p/managedZones/z1:setIamPolicy",
		`{"policy":{"version":1,"bindings":[{"role":"roles/dns.admin","members":["user:dev@localhost"]}]}}`)
	if code != 200 {
		t.Fatalf("setIamPolicy: %d %v", code, pol)
	}
	if pol["etag"] == "" {
		t.Fatal("etag missing")
	}

	// get policy
	code, got := do(t, h, "POST", "/dns/v1/projects/p/managedZones/z1:getIamPolicy", `{}`)
	if code != 200 {
		t.Fatalf("getIamPolicy: %d %v", code, got)
	}
	bindings, _ := got["bindings"].([]any)
	if len(bindings) != 1 {
		t.Fatalf("unexpected bindings: %v", got)
	}

	// test permissions: admin allows all
	code, tp := do(t, h, "POST", "/dns/v1/projects/p/managedZones/z1:testIamPermissions",
		`{"permissions":["dns.managedZones.get","dns.managedZones.delete"]}`)
	if code != 200 {
		t.Fatalf("testIamPermissions: %d %v", code, tp)
	}
	perms, _ := tp["permissions"].([]any)
	if len(perms) != 2 {
		t.Fatalf("expected 2 permissions, got %v", tp)
	}
}

func TestDNSUnimplementedAndNotFound(t *testing.T) {
	h := newTestHandler(t)
	code, body := do(t, h, "GET", "/dns/v1/projects/p/managedZones/z/dnsKeys", "")
	if code != 501 {
		t.Fatalf("dnsKeys: got %d, want 501 (%v)", code, body)
	}
	code, _ = do(t, h, "GET", "/dns/v1/projects/p/managedZones/z/operations", "")
	if code != 200 {
		t.Fatalf("zone operations list: got %d", code)
	}
	code, _ = do(t, h, "GET", "/dns/v1/projects/p/operations", "")
	if code != 200 {
		t.Fatalf("project operations list: got %d", code)
	}
	code, body = do(t, h, "GET", "/dns/v1/projects/p/operations/abc", "")
	if code != 404 {
		t.Fatalf("operations.get: got %d, want 404 (%v)", code, body)
	}
	code, _ = do(t, h, "GET", "/dns/v1/projects/p/managedZones/missing", "")
	if code != 404 {
		t.Fatalf("missing zone: got %d, want 404", code)
	}
}

func TestDNSChangeValidation(t *testing.T) {
	h := newTestHandler(t)
	if code, _ := do(t, h, "POST", "/dns/v1/projects/p/managedZones",
		`{"name":"z","dnsName":"example.com."}`); code != 200 {
		t.Fatalf("create zone: %d", code)
	}
	// rrset outside zone must fail
	code, body := do(t, h, "POST", "/dns/v1/projects/p/managedZones/z/changes",
		`{"additions":[{"name":"www.other.com.","type":"A","ttl":300,"rrdatas":["192.0.2.1"]}]}`)
	if code != 400 {
		t.Fatalf("out-of-zone rrset: got %d, want 400 (%v)", code, body)
	}
	// invalid zone name
	code, _ = do(t, h, "POST", "/dns/v1/projects/p/managedZones", `{"name":"Bad_Name","dnsName":"x.com."}`)
	if code != 400 {
		t.Fatalf("invalid zone name: got %d, want 400", code)
	}
}
