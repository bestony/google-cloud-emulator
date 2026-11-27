package iam

import "testing"

func TestSetGetRoundTrip(t *testing.T) {
	s := NewStore()
	p := s.Set("projects/p", Policy{Bindings: []Binding{
		{Role: "roles/owner", Members: []string{"user:a@example.com"}},
	}})
	if p.Etag == "" {
		t.Fatal("etag not set")
	}
	got := s.Get("projects/p")
	if got.Version != 1 || len(got.Bindings) != 1 || got.Bindings[0].Role != "roles/owner" {
		t.Fatalf("unexpected policy: %+v", got)
	}
	if got.Etag != p.Etag {
		t.Fatal("etag mismatch")
	}
}

func TestTestPermissionsHeuristic(t *testing.T) {
	s := NewStore()
	s.Set("projects/p", Policy{Bindings: []Binding{
		{Role: "roles/editor", Members: []string{"user:dev@localhost"}},
		{Role: "roles/viewer", Members: []string{"user:viewer@example.com"}},
	}})
	perms := []string{"secretmanager.secrets.get", "secretmanager.secrets.create", "dns.managedZones.list"}

	got := s.Test("projects/p", perms, "dev@localhost")
	if len(got) != 3 {
		t.Fatalf("editor should allow all, got %v", got)
	}
	got = s.Test("projects/p", perms, "viewer@example.com")
	want := map[string]bool{"secretmanager.secrets.get": true, "dns.managedZones.list": true}
	if len(got) != 2 {
		t.Fatalf("viewer should allow reads only, got %v", got)
	}
	for _, p := range got {
		if !want[p] {
			t.Fatalf("unexpected permission %q", p)
		}
	}
	if got := s.Test("projects/p", perms, "nobody@example.com"); len(got) != 0 {
		t.Fatalf("unknown user should get nothing, got %v", got)
	}
	if got := s.Test("projects/other", perms, "dev@localhost"); len(got) != 0 {
		t.Fatalf("unbound resource should get nothing, got %v", got)
	}
}
