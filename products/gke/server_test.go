package gke

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

	containerpb "github.com/bestony/google-cloud-emulator/gen/google/container/v1"
)

const project = "demo"

func createCluster(t *testing.T, s *server, zone, name string) *containerpb.Operation {
	t.Helper()
	op, err := s.CreateCluster(context.Background(), &containerpb.CreateClusterRequest{
		ProjectId: project,
		Zone:      zone,
		Cluster:   &containerpb.Cluster{Name: name, InitialNodeCount: 3},
	})
	if err != nil {
		t.Fatalf("CreateCluster %s: %v", name, err)
	}
	return op
}

func TestClusterLifecycle(t *testing.T) {
	st := NewState()
	s := &server{state: st}
	ctx := context.Background()

	op := createCluster(t, s, "us-central1-a", "c1")
	if op.GetStatus() != containerpb.Operation_DONE ||
		op.GetOperationType() != containerpb.Operation_CREATE_CLUSTER {
		t.Fatalf("unexpected operation: %+v", op)
	}
	if op.GetName() == "" || !strings.HasPrefix(op.GetName(), "operation-") ||
		op.GetSelfLink() == "" || op.GetTargetLink() == "" {
		t.Fatalf("operation missing fields: %+v", op)
	}

	// Duplicate -> AlreadyExists.
	if _, err := s.CreateCluster(ctx, &containerpb.CreateClusterRequest{
		ProjectId: project, Zone: "us-central1-a",
		Cluster: &containerpb.Cluster{Name: "c1"},
	}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate create: got %v", err)
	}

	c, err := s.GetCluster(ctx, &containerpb.GetClusterRequest{
		ProjectId: project, Zone: "us-central1-a", ClusterId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCluster: %v", err)
	}
	if c.GetName() != "c1" || c.GetStatus() != containerpb.Cluster_RUNNING ||
		c.GetSelfLink() == "" || c.GetId() == "" || c.GetEtag() == "" ||
		c.GetCreateTime() == "" || c.GetLocation() != "us-central1-a" {
		t.Fatalf("unexpected cluster: %+v", c)
	}
	if c.GetInitialClusterVersion() == "" || c.GetCurrentMasterVersion() == "" {
		t.Fatalf("cluster missing versions: %+v", c)
	}
	// Full resource name also resolves.
	c2, err := s.GetCluster(ctx, &containerpb.GetClusterRequest{
		Name: "projects/" + project + "/locations/us-central1-a/clusters/c1",
	})
	if err != nil || c2.GetName() != "c1" {
		t.Fatalf("GetCluster by resource name: %v %+v", err, c2)
	}

	// Second cluster in another zone; list filters by zone.
	createCluster(t, s, "europe-west1-b", "c2")
	l, err := s.ListClusters(ctx, &containerpb.ListClustersRequest{ProjectId: project, Zone: "us-central1-a"})
	if err != nil || len(l.GetClusters()) != 1 || l.GetClusters()[0].GetName() != "c1" {
		t.Fatalf("ListClusters zone filter: %v %+v", err, l)
	}
	lAll, err := s.ListClusters(ctx, &containerpb.ListClustersRequest{ProjectId: project})
	if err != nil || len(lAll.GetClusters()) != 2 {
		t.Fatalf("ListClusters all: %v %+v", err, lAll)
	}
	// Parent form.
	lParent, err := s.ListClusters(ctx, &containerpb.ListClustersRequest{
		Parent: "projects/" + project + "/locations/europe-west1-b",
	})
	if err != nil || len(lParent.GetClusters()) != 1 || lParent.GetClusters()[0].GetName() != "c2" {
		t.Fatalf("ListClusters parent: %v %+v", err, lParent)
	}

	// Update: desired node version -> UPGRADE_MASTER operation.
	uop, err := s.UpdateCluster(ctx, &containerpb.UpdateClusterRequest{
		ProjectId: project, Zone: "us-central1-a", ClusterId: "c1",
		Update: &containerpb.ClusterUpdate{DesiredNodeVersion: "1.32.0-gke.1000000"},
	})
	if err != nil || uop.GetOperationType() != containerpb.Operation_UPGRADE_MASTER {
		t.Fatalf("UpdateCluster: %v %+v", err, uop)
	}
	cu, _ := s.GetCluster(ctx, &containerpb.GetClusterRequest{ProjectId: project, Zone: "us-central1-a", ClusterId: "c1"})
	if cu.GetCurrentMasterVersion() != "1.32.0-gke.1000000" {
		t.Fatalf("master version not updated: %+v", cu)
	}

	// Delete.
	if _, err := s.DeleteCluster(ctx, &containerpb.DeleteClusterRequest{
		ProjectId: project, Zone: "us-central1-a", ClusterId: "c1",
	}); err != nil {
		t.Fatalf("DeleteCluster: %v", err)
	}
	if _, err := s.GetCluster(ctx, &containerpb.GetClusterRequest{ProjectId: project, Zone: "us-central1-a", ClusterId: "c1"}); status.Code(err) != codes.NotFound {
		t.Fatalf("get deleted cluster: got %v", err)
	}
}

func TestOperations(t *testing.T) {
	st := NewState()
	s := &server{state: st}
	ctx := context.Background()

	op := createCluster(t, s, "us-central1-a", "c1")

	got, err := s.GetOperation(ctx, &containerpb.GetOperationRequest{
		ProjectId: project, Zone: "us-central1-a", OperationId: op.GetName(),
	})
	if err != nil || got.GetName() != op.GetName() {
		t.Fatalf("GetOperation: %v %+v", err, got)
	}
	// Full resource name resolves too.
	got2, err := s.GetOperation(ctx, &containerpb.GetOperationRequest{
		Name: "projects/" + project + "/locations/us-central1-a/operations/" + op.GetName(),
	})
	if err != nil || got2.GetName() != op.GetName() {
		t.Fatalf("GetOperation by resource name: %v %+v", err, got2)
	}
	if _, err := s.GetOperation(ctx, &containerpb.GetOperationRequest{
		ProjectId: project, Zone: "us-central1-a", OperationId: "operation-999",
	}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing operation: got %v", err)
	}

	l, err := s.ListOperations(ctx, &containerpb.ListOperationsRequest{ProjectId: project, Zone: "us-central1-a"})
	if err != nil || len(l.GetOperations()) != 1 {
		t.Fatalf("ListOperations: %v %+v", err, l)
	}
	l2, err := s.ListOperations(ctx, &containerpb.ListOperationsRequest{
		Parent: "projects/" + project + "/locations/us-central1-a",
	})
	if err != nil || len(l2.GetOperations()) != 1 {
		t.Fatalf("ListOperations parent: %v %+v", err, l2)
	}
	createCluster(t, s, "europe-west1-b", "c2")
	l3, err := s.ListOperations(ctx, &containerpb.ListOperationsRequest{ProjectId: project})
	if err != nil || len(l3.GetOperations()) != 2 {
		t.Fatalf("ListOperations all zones: %v %+v", err, l3)
	}

	// Node pool / server config RPCs stay honestly UNIMPLEMENTED.
	if _, err := s.GetNodePool(ctx, &containerpb.GetNodePoolRequest{}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("GetNodePool: got %v, want Unimplemented", err)
	}
	if _, err := s.GetServerConfig(ctx, &containerpb.GetServerConfigRequest{}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("GetServerConfig: got %v, want Unimplemented", err)
	}
}

func TestGRPCTransport(t *testing.T) {
	st := NewState()
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

	cc := containerpb.NewClusterManagerClient(conn)
	if _, err := cc.CreateCluster(context.Background(), &containerpb.CreateClusterRequest{
		ProjectId: project, Zone: "us-central1-a",
		Cluster: &containerpb.Cluster{Name: "grpc-c"},
	}); err != nil {
		t.Fatalf("gRPC CreateCluster: %v", err)
	}
	got, err := cc.GetCluster(context.Background(), &containerpb.GetClusterRequest{
		ProjectId: project, Zone: "us-central1-a", ClusterId: "grpc-c",
	})
	if err != nil || got.GetName() != "grpc-c" {
		t.Fatalf("gRPC GetCluster: %v %+v", err, got)
	}
}

func TestRESTGateway(t *testing.T) {
	st := NewState()
	mux := runtime.NewServeMux()
	if err := Register(grpc.NewServer(), mux, st); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(WrapGateway(mux))
	defer ts.Close()

	// Create via legacy zones REST path.
	req, _ := http.NewRequest("POST",
		ts.URL+"/v1/projects/"+project+"/zones/us-central1-a/clusters",
		strings.NewReader(`{"name":"rest-c","initialNodeCount":2}`))
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
	var op map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&op); err != nil {
		t.Fatal(err)
	}
	if op["status"] != "DONE" || op["name"] == "" {
		t.Fatalf("unexpected operation: %v", op)
	}

	// Get via locations REST path.
	resp2, err := http.Get(ts.URL + "/v1/projects/" + project + "/locations/us-central1-a/clusters/rest-c")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var c map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&c); err != nil {
		t.Fatal(err)
	}
	if c["name"] != "rest-c" || c["status"] != "RUNNING" {
		t.Fatalf("unexpected cluster: %v", c)
	}

	// List clusters.
	resp3, err := http.Get(ts.URL + "/v1/projects/" + project + "/locations/us-central1-a/clusters")
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	var lc map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&lc); err != nil {
		t.Fatal(err)
	}
	if _, ok := lc["clusters"].([]any); !ok {
		t.Fatalf("unexpected list response: %v", lc)
	}

	// List operations.
	resp4, err := http.Get(ts.URL + "/v1/projects/" + project + "/locations/us-central1-a/operations")
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	var lo map[string]any
	if err := json.NewDecoder(resp4.Body).Decode(&lo); err != nil {
		t.Fatal(err)
	}
	ops, _ := lo["operations"].([]any)
	if len(ops) != 1 {
		t.Fatalf("unexpected operations list: %v", lo)
	}

	// Update via PUT with a bare ClusterUpdate body (shim wraps it).
	reqUpd, _ := http.NewRequest("PUT",
		ts.URL+"/v1/projects/"+project+"/zones/us-central1-a/clusters/rest-c",
		strings.NewReader(`{"desiredNodeVersion":"1.32.1-gke.1000000"}`))
	reqUpd.Header.Set("Content-Type", "application/json")
	respUpd, err := http.DefaultClient.Do(reqUpd)
	if err != nil {
		t.Fatal(err)
	}
	defer respUpd.Body.Close()
	if respUpd.StatusCode != 200 {
		raw, _ := io.ReadAll(respUpd.Body)
		t.Fatalf("update status = %d body=%s", respUpd.StatusCode, raw)
	}
	var uop map[string]any
	if err := json.NewDecoder(respUpd.Body).Decode(&uop); err != nil {
		t.Fatal(err)
	}
	if uop["operationType"] != "UPGRADE_MASTER" {
		t.Fatalf("unexpected update operation: %v", uop)
	}

	// Delete via locations path.
	req5, _ := http.NewRequest("DELETE",
		ts.URL+"/v1/projects/"+project+"/locations/us-central1-a/clusters/rest-c", nil)
	resp5, err := http.DefaultClient.Do(req5)
	if err != nil {
		t.Fatal(err)
	}
	defer resp5.Body.Close()
	if resp5.StatusCode != 200 {
		raw, _ := io.ReadAll(resp5.Body)
		t.Fatalf("delete status = %d body=%s", resp5.StatusCode, raw)
	}

	// Scaffolded method on the same service stays UNIMPLEMENTED.
	resp6, err := http.Get(ts.URL + "/v1/projects/" + project + "/locations/us-central1-a/clusters/rest-c/nodePools")
	if err != nil {
		t.Fatal(err)
	}
	defer resp6.Body.Close()
	if resp6.StatusCode != 501 {
		raw, _ := io.ReadAll(resp6.Body)
		t.Fatalf("nodePools status = %d body=%s, want 501", resp6.StatusCode, raw)
	}
}
