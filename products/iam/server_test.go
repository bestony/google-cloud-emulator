package iam

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	coreiam "github.com/bestony/google-cloud-emulator/core/iam"
	iampb "github.com/bestony/google-cloud-emulator/gen/google/iam/v1"
	typeexpr "github.com/bestony/google-cloud-emulator/gen/google/type"
)

func newServer() (*server, *coreiam.Store) {
	store := coreiam.NewStore()
	return &server{state: NewState(store)}, store
}

func TestSetGetTestIamPolicy(t *testing.T) {
	s, _ := newServer()
	ctx := context.Background()
	resource := "//secretmanager.googleapis.com/projects/demo/secrets/s1"

	// Empty policy before any set.
	got, err := s.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: resource})
	if err != nil {
		t.Fatalf("GetIamPolicy: %v", err)
	}
	if len(got.GetBindings()) != 0 {
		t.Fatalf("expected empty policy, got %+v", got)
	}

	// Set a policy.
	set, err := s.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: resource,
		Policy: &iampb.Policy{
			Version: 1,
			Bindings: []*iampb.Binding{
				{Role: "roles/secretmanager.admin", Members: []string{"user:alice@example.com"}},
				{Role: "roles/viewer", Members: []string{"allUsers"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("SetIamPolicy: %v", err)
	}
	if len(set.GetEtag()) == 0 || len(set.GetBindings()) != 2 {
		t.Fatalf("unexpected policy: %+v", set)
	}

	// Get returns the same bindings; host prefix normalized.
	got2, err := s.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: "projects/demo/secrets/s1"})
	if err != nil {
		t.Fatalf("GetIamPolicy relative: %v", err)
	}
	if len(got2.GetBindings()) != 2 || !bytes.Equal(got2.GetEtag(), set.GetEtag()) {
		t.Fatalf("policy mismatch: %+v vs %+v", got2, set)
	}

	// TestIamPermissions: caller matches allUsers -> viewer read perms.
	mdCtx := metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "Bearer token"))
	resp, err := s.TestIamPermissions(mdCtx, &iampb.TestIamPermissionsRequest{
		Resource:    resource,
		Permissions: []string{"secretmanager.secrets.get", "secretmanager.secrets.setIamPolicy"},
	})
	if err != nil {
		t.Fatalf("TestIamPermissions: %v", err)
	}
	// dev@localhost is in allAuthenticatedUsers/allUsers? callerMembers includes
	// allUsers unconditionally, and the viewer binding grants read-only .get.
	if len(resp.GetPermissions()) != 1 || resp.GetPermissions()[0] != "secretmanager.secrets.get" {
		t.Fatalf("unexpected permissions: %+v", resp.GetPermissions())
	}
}

func TestHonestUnsupported(t *testing.T) {
	s, _ := newServer()
	ctx := context.Background()

	// Conditions are not modeled -> Unimplemented, never silently dropped.
	_, err := s.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: "projects/p",
		Policy: &iampb.Policy{Bindings: []*iampb.Binding{{
			Role:      "roles/viewer",
			Members:   []string{"allUsers"},
			Condition: &typeexpr.Expr{Expression: "request.time < timestamp('2020-01-01T00:00:00Z')"},
		}}},
	})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("condition binding: got %v, want Unimplemented", err)
	}

	// Audit configs are not modeled either.
	_, err = s.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: "projects/p",
		Policy: &iampb.Policy{
			Bindings:     []*iampb.Binding{{Role: "roles/viewer", Members: []string{"allUsers"}}},
			AuditConfigs: []*iampb.AuditConfig{{Service: "allServices"}},
		},
	})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("audit configs: got %v, want Unimplemented", err)
	}
}

func TestValidation(t *testing.T) {
	s, _ := newServer()
	ctx := context.Background()
	if _, err := s.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty resource: got %v", err)
	}
	if _, err := s.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{Resource: "projects/p"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing policy: got %v", err)
	}
}

func TestGRPCTransport(t *testing.T) {
	store := coreiam.NewStore()
	st := NewState(store)
	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	if err := Register(gs, runtime.NewServeMux(), st); err != nil {
		t.Fatal(err)
	}
	go func() { _ = gs.Serve(lis) }()
	defer gs.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	c := iampb.NewIAMPolicyClient(conn)
	if _, err := c.SetIamPolicy(context.Background(), &iampb.SetIamPolicyRequest{
		Resource: "projects/demo",
		Policy:   &iampb.Policy{Bindings: []*iampb.Binding{{Role: "roles/owner", Members: []string{"user:dev@localhost"}}}},
	}); err != nil {
		t.Fatalf("gRPC SetIamPolicy: %v", err)
	}
	got, err := c.GetIamPolicy(context.Background(), &iampb.GetIamPolicyRequest{Resource: "projects/demo"})
	if err != nil || len(got.GetBindings()) != 1 {
		t.Fatalf("gRPC GetIamPolicy: %v %+v", err, got)
	}
	tresp, err := c.TestIamPermissions(context.Background(), &iampb.TestIamPermissionsRequest{
		Resource: "projects/demo", Permissions: []string{"resourcemanager.projects.delete"},
	})
	if err != nil || len(tresp.GetPermissions()) != 1 {
		t.Fatalf("gRPC TestIamPermissions: %v %+v", err, tresp)
	}
}

func TestRESTGateway(t *testing.T) {
	store := coreiam.NewStore()
	st := NewState(store)
	mux := runtime.NewServeMux()
	if err := Register(grpc.NewServer(), mux, st); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resource := "projects/demo/secrets/s1"
	body := `{"policy":{"version":1,"bindings":[{"role":"roles/secretmanager.admin","members":["user:alice@example.com"]}]}}`
	req, _ := http.NewRequest("POST", ts.URL+"/v1/"+resource+":setIamPolicy", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("setIamPolicy status = %d body=%s", resp.StatusCode, raw)
	}
	var pol map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&pol); err != nil {
		t.Fatal(err)
	}
	if pol["etag"] == nil || pol["etag"] == "" {
		t.Fatalf("missing etag: %v", pol)
	}

	// getIamPolicy via REST.
	req2, _ := http.NewRequest("POST", ts.URL+"/v1/"+resource+":getIamPolicy", strings.NewReader(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var got map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	bindings, _ := got["bindings"].([]any)
	if len(bindings) != 1 {
		t.Fatalf("unexpected bindings: %v", got)
	}

	// testIamPermissions via REST.
	req3, _ := http.NewRequest("POST", ts.URL+"/v1/"+resource+":testIamPermissions",
		strings.NewReader(`{"permissions":["secretmanager.secrets.get"]}`))
	req3.Header.Set("Content-Type", "application/json")
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	var tp map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&tp); err != nil {
		t.Fatal(err)
	}
	perms, _ := tp["permissions"].([]any)
	if len(perms) != 0 {
		t.Fatalf("dev@localhost should have no permissions here, got %v", tp)
	}
}
