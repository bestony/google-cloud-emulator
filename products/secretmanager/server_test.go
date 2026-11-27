package secretmanager

import (
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
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bestony/google-cloud-emulator/core/iam"
	smpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/secretmanager/v1"
	iampb "github.com/bestony/google-cloud-emulator/gen/google/iam/v1"
)

func newTestServer(t *testing.T) (*Server, *State) {
	t.Helper()
	st := NewState(iam.NewStore())
	return NewServer(st), st
}

func TestSecretLifecycle(t *testing.T) {
	srv, _ := newTestServer(t)
	ctx := context.Background()

	sec, err := srv.CreateSecret(ctx, &smpb.CreateSecretRequest{
		Parent:   "projects/test-project",
		SecretId: "my-secret",
		Secret: &smpb.Secret{
			Labels: map[string]string{"env": "dev"},
		},
	})
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if sec.Name != "projects/test-project/secrets/my-secret" {
		t.Fatalf("unexpected name %q", sec.Name)
	}
	if sec.State != smpb.Secret_STATE_ACTIVE {
		t.Fatalf("unexpected state %v", sec.State)
	}
	if sec.Etag == "" {
		t.Fatal("etag not set")
	}

	// Duplicate create must fail with ALREADY_EXISTS.
	_, err = srv.CreateSecret(ctx, &smpb.CreateSecretRequest{
		Parent: "projects/test-project", SecretId: "my-secret", Secret: &smpb.Secret{},
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate create: got %v", err)
	}

	v, err := srv.AddSecretVersion(ctx, &smpb.AddSecretVersionRequest{
		Parent:  sec.Name,
		Payload: &smpb.SecretPayload{Data: []byte("s3cr3t")},
	})
	if err != nil {
		t.Fatalf("AddSecretVersion: %v", err)
	}
	if v.State != smpb.SecretVersion_ENABLED {
		t.Fatalf("unexpected version state %v", v.State)
	}

	got, err := srv.GetSecret(ctx, &smpb.GetSecretRequest{Name: sec.Name})
	if err != nil || got.Name != sec.Name {
		t.Fatalf("GetSecret: %v %+v", err, got)
	}

	access, err := srv.AccessSecretVersion(ctx, &smpb.AccessSecretVersionRequest{
		Name: sec.Name + "/versions/latest",
	})
	if err != nil {
		t.Fatalf("AccessSecretVersion(latest): %v", err)
	}
	if string(access.Payload.Data) != "s3cr3t" {
		t.Fatalf("unexpected payload %q", access.Payload.Data)
	}
	if access.Payload.DataCrc32C == nil {
		t.Fatal("checksum missing")
	}

	// Disable then access must fail.
	if _, err := srv.DisableSecretVersion(ctx, &smpb.DisableSecretVersionRequest{Name: v.Name}); err != nil {
		t.Fatalf("DisableSecretVersion: %v", err)
	}
	if _, err := srv.AccessSecretVersion(ctx, &smpb.AccessSecretVersionRequest{Name: v.Name}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("access disabled: got %v", err)
	}
	if _, err := srv.EnableSecretVersion(ctx, &smpb.EnableSecretVersionRequest{Name: v.Name}); err != nil {
		t.Fatalf("EnableSecretVersion: %v", err)
	}

	// Update labels via mask.
	updated, err := srv.UpdateSecret(ctx, &smpb.UpdateSecretRequest{
		Secret:     &smpb.Secret{Name: sec.Name, Labels: map[string]string{"env": "prod"}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
	})
	if err != nil {
		t.Fatalf("UpdateSecret: %v", err)
	}
	if updated.Labels["env"] != "prod" {
		t.Fatalf("labels not updated: %v", updated.Labels)
	}

	// List versions.
	lv, err := srv.ListSecretVersions(ctx, &smpb.ListSecretVersionsRequest{Parent: sec.Name})
	if err != nil || len(lv.Versions) != 1 {
		t.Fatalf("ListSecretVersions: %v %+v", err, lv)
	}

	// Destroy version, then access must fail.
	if _, err := srv.DestroySecretVersion(ctx, &smpb.DestroySecretVersionRequest{Name: v.Name}); err != nil {
		t.Fatalf("DestroySecretVersion: %v", err)
	}
	if _, err := srv.AccessSecretVersion(ctx, &smpb.AccessSecretVersionRequest{Name: v.Name}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("access destroyed: got %v", err)
	}

	// Delete secret; subsequent get must be NOT_FOUND.
	if _, err := srv.DeleteSecret(ctx, &smpb.DeleteSecretRequest{Name: sec.Name}); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	if _, err := srv.GetSecret(ctx, &smpb.GetSecretRequest{Name: sec.Name}); status.Code(err) != codes.NotFound {
		t.Fatalf("get deleted: got %v", err)
	}
}

func TestListSecretsPagination(t *testing.T) {
	srv, _ := newTestServer(t)
	ctx := context.Background()
	for _, id := range []string{"a-secret", "b-secret", "c-secret"} {
		if _, err := srv.CreateSecret(ctx, &smpb.CreateSecretRequest{
			Parent: "projects/p", SecretId: id, Secret: &smpb.Secret{},
		}); err != nil {
			t.Fatal(err)
		}
	}
	page1, err := srv.ListSecrets(ctx, &smpb.ListSecretsRequest{Parent: "projects/p", PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Secrets) != 2 || page1.NextPageToken == "" {
		t.Fatalf("unexpected page1: %+v", page1)
	}
	page2, err := srv.ListSecrets(ctx, &smpb.ListSecretsRequest{Parent: "projects/p", PageSize: 2, PageToken: page1.NextPageToken})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Secrets) != 1 || page2.NextPageToken != "" {
		t.Fatalf("unexpected page2: %+v", page2)
	}
	if page1.Secrets[0].Name >= page1.Secrets[1].Name {
		t.Fatal("secrets not sorted")
	}
}

func TestIAMPolicy(t *testing.T) {
	srv, _ := newTestServer(t)
	ctx := context.Background()
	sec, err := srv.CreateSecret(ctx, &smpb.CreateSecretRequest{
		Parent: "projects/p", SecretId: "s", Secret: &smpb.Secret{},
	})
	if err != nil {
		t.Fatal(err)
	}
	pol, err := srv.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: sec.Name,
		Policy: &iampb.Policy{
			Bindings: []*iampb.Binding{{Role: "roles/secretmanager.admin", Members: []string{"user:dev@localhost"}}},
		},
	})
	if err != nil {
		t.Fatalf("SetIamPolicy: %v", err)
	}
	if pol.Etag == nil || len(pol.Bindings) != 1 {
		t.Fatalf("unexpected policy: %+v", pol)
	}
	got, err := srv.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: sec.Name})
	if err != nil || got.Bindings[0].Role != "roles/secretmanager.admin" {
		t.Fatalf("GetIamPolicy: %v %+v", err, got)
	}
	// TestIamPermissions for the caller matches "user:dev@localhost" (admin => all).
	tp, err := srv.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
		Resource:    sec.Name,
		Permissions: []string{"secretmanager.secrets.get", "secretmanager.secrets.destroy"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tp.Permissions) != 2 {
		t.Fatalf("expected all permissions, got %v", tp.Permissions)
	}
	// Missing secret yields empty permissions, not NOT_FOUND.
	tp, err = srv.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
		Resource:    "projects/p/secrets/missing",
		Permissions: []string{"secretmanager.secrets.get"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tp.Permissions) != 0 {
		t.Fatalf("expected empty permissions, got %v", tp.Permissions)
	}
}

func TestGRPCTransport(t *testing.T) {
	srv, _ := newTestServer(t)
	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	smpb.RegisterSecretManagerServiceServer(gs, srv)
	go func() { _ = gs.Serve(lis) }()
	defer gs.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := smpb.NewSecretManagerServiceClient(conn)

	sec, err := client.CreateSecret(context.Background(), &smpb.CreateSecretRequest{
		Parent: "projects/p", SecretId: "via-grpc", Secret: &smpb.Secret{},
	})
	if err != nil {
		t.Fatalf("gRPC CreateSecret: %v", err)
	}
	if _, err := client.AddSecretVersion(context.Background(), &smpb.AddSecretVersionRequest{
		Parent: sec.Name, Payload: &smpb.SecretPayload{Data: []byte("v")},
	}); err != nil {
		t.Fatalf("gRPC AddSecretVersion: %v", err)
	}
	resp, err := client.AccessSecretVersion(context.Background(), &smpb.AccessSecretVersionRequest{Name: sec.Name + "/versions/1"})
	if err != nil {
		t.Fatalf("gRPC AccessSecretVersion: %v", err)
	}
	if string(resp.Payload.Data) != "v" {
		t.Fatalf("unexpected payload %q", resp.Payload.Data)
	}
}

func TestRESTGateway(t *testing.T) {
	st := NewState(iam.NewStore())
	srv := NewServer(st)
	mux := runtime.NewServeMux()
	if err := smpb.RegisterSecretManagerServiceHandlerServer(context.Background(), mux, srv); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Create via REST: POST /v1/{parent=projects/*}/secrets?secretId=...
	body := `{"labels":{"team":"platform"}}`
	req, err := http.NewRequest("POST", ts.URL+"/v1/projects/p/secrets?secretId=rest-secret", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("create status = %d body=%s", resp.StatusCode, raw)
	}
	var created struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Name != "projects/p/secrets/rest-secret" || created.Labels["team"] != "platform" {
		t.Fatalf("unexpected created secret: %+v", created)
	}

	// Access version via REST after adding one through the gRPC server.
	if _, err := srv.AddSecretVersion(context.Background(), &smpb.AddSecretVersionRequest{
		Parent: created.Name, Payload: &smpb.SecretPayload{Data: []byte("rest-data")},
	}); err != nil {
		t.Fatal(err)
	}
	resp2, err := http.Get(ts.URL + "/v1/" + created.Name + "/versions/latest:access")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		raw, _ := io.ReadAll(resp2.Body)
		t.Fatalf("access status = %d body=%s", resp2.StatusCode, raw)
	}
	var access struct {
		Payload struct {
			Data string `json:"data"`
		} `json:"payload"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&access); err != nil {
		t.Fatal(err)
	}
	// REST JSON encodes bytes as base64.
	if access.Payload.Data != "cmVzdC1kYXRh" {
		t.Fatalf("unexpected payload %q", access.Payload.Data)
	}
}
