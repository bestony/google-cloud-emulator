package lro

import (
	"testing"

	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
)

func TestOperationLifecycle(t *testing.T) {
	s := NewStore()
	op := s.Start("projects/p/operations", "dns")
	if op.Name != "projects/p/operations/1" {
		t.Fatalf("unexpected name %q", op.Name)
	}
	if op.Done {
		t.Fatal("new operation should not be done")
	}
	if _, ok := s.Get(op.Name); !ok {
		t.Fatal("operation not found")
	}
	s.Complete(op, map[string]string{"ok": "yes"})
	if !op.Done || op.Result == nil {
		t.Fatal("operation not completed")
	}
	listed := s.List("projects/p/operations")
	if len(listed) != 1 {
		t.Fatalf("list returned %d ops", len(listed))
	}
	s.Fail(s.Start("projects/p/operations", "dns"), gcperr.Internal, "boom")
	if len(s.List("projects/p/operations")) != 2 {
		t.Fatal("expected 2 operations")
	}
	if len(s.List("projects/q/operations")) != 0 {
		t.Fatal("prefix filter failed")
	}
}
