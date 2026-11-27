package run

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	runpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/run/v2"
	iampb "github.com/bestony/google-cloud-emulator/gen/google/iam/v1"
	"github.com/bestony/google-cloud-emulator/core/iam"
	"github.com/bestony/google-cloud-emulator/core/lro"
)

func newTestServer(t *testing.T) (*Server, *lro.Store) {
	t.Helper()
	ls := lro.NewStore()
	return NewServer(NewState(ls, iam.NewStore())), ls
}

const (
	location = "projects/demo/locations/us-central1"
	svcName = location + "/services/web"
	rev1    = svcName + "/revisions/web-00002"
)

func TestServiceLifecycle(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	op, err := srv.CreateService(ctx, &runpb.CreateServiceRequest{
		Parent:    location,
		ServiceId: "web",
		Service: &runpb.Service{
			Labels: map[string]string{"env": "dev"},
			Template: &runpb.RevisionTemplate{
				Containers: []*runpb.Container{{Image: "us-docker.pkg.dev/demo/app:latest"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	if !op.Done {
		t.Fatalf("operation not done: %+v", op)
	}
	resp, err := op.GetResponse().UnmarshalNew()
	if err != nil {
		t.Fatal(err)
	}
	svc := resp.(*runpb.Service)
	if svc.Name != svcName || svc.Uid == "" || svc.Generation != 1 {
		t.Fatalf("unexpected service: %+v", svc)
	}
	if svc.LatestReadyRevision == "" || svc.Uri == "" {
		t.Fatalf("missing revision/uri: %+v", svc)
	}
	if _, ok := store.Get(op.Name); !ok {
		t.Fatalf("operation %q not in store", op.Name)
	}

	// Duplicate -> AlreadyExists.
	if _, err := srv.CreateService(ctx, &runpb.CreateServiceRequest{Parent: location, ServiceId: "web", Service: &runpb.Service{}}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate: got %v", err)
	}

	// Get + List.
	got, err := srv.GetService(ctx, &runpb.GetServiceRequest{Name: svcName})
	if err != nil || got.Labels["env"] != "dev" {
		t.Fatalf("GetService: %v %+v", err, got)
	}
	ls, err := srv.ListServices(ctx, &runpb.ListServicesRequest{Parent: location})
	if err != nil || len(ls.Services) != 1 {
		t.Fatalf("ListServices: %v %+v", err, ls)
	}

	// Revision exists and is ready.
	rev, err := srv.GetRevision(ctx, &runpb.GetRevisionRequest{Name: svc.LatestReadyRevision})
	if err != nil {
		t.Fatalf("GetRevision: %v", err)
	}
	if rev.Service != svcName || len(rev.Conditions) == 0 || len(rev.Containers) != 1 || rev.Containers[0].Image == "" {
		t.Fatalf("unexpected revision: %+v", rev)
	}
	lr, err := srv.ListRevisions(ctx, &runpb.ListRevisionsRequest{Parent: location})
	if err != nil || len(lr.Revisions) != 1 {
		t.Fatalf("ListRevisions: %v %+v", err, lr)
	}

	// Update labels + template -> new revision, generation bumps.
	uop, err := srv.UpdateService(ctx, &runpb.UpdateServiceRequest{
		Service: &runpb.Service{
			Name:   svcName,
			Labels: map[string]string{"env": "prod"},
			Template: &runpb.RevisionTemplate{
				Containers: []*runpb.Container{{Image: "us-docker.pkg.dev/demo/app:v2"}},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels", "template"}},
	})
	if err != nil {
		t.Fatalf("UpdateService: %v", err)
	}
	uresp, _ := uop.GetResponse().UnmarshalNew()
	usvc := uresp.(*runpb.Service)
	if usvc.Generation != 2 || usvc.Labels["env"] != "prod" {
		t.Fatalf("unexpected updated service: %+v", usvc)
	}
	if usvc.LatestReadyRevision == svc.LatestReadyRevision {
		t.Fatal("expected a new revision after template update")
	}
	lrs, _ := srv.ListRevisions(ctx, &runpb.ListRevisionsRequest{Parent: svcName})
	if len(lrs.Revisions) != 2 {
		t.Fatalf("expected 2 revisions, got %d", len(lrs.Revisions))
	}

	// Bad mask -> InvalidArgument.
	if _, err := srv.UpdateService(ctx, &runpb.UpdateServiceRequest{
		Service:    &runpb.Service{Name: svcName},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("bad mask: got %v", err)
	}

	// Delete -> LRO, service gone, revisions gone.
	dop, err := srv.DeleteService(ctx, &runpb.DeleteServiceRequest{Name: svcName})
	if err != nil || !dop.Done {
		t.Fatalf("DeleteService: %v %+v", err, dop)
	}
	if _, err := srv.GetService(ctx, &runpb.GetServiceRequest{Name: svcName}); err == nil {
		t.Fatal("service still exists after delete")
	}
	if _, err := srv.GetRevision(ctx, &runpb.GetRevisionRequest{Name: rev1}); err == nil {
		t.Fatal("revision still exists after service delete")
	}
}

func TestIAM(t *testing.T) {
	srv, _ := newTestServer(t)
	ctx := context.Background()
	res := svcName

	pol, err := srv.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: res,
		Policy: &iampb.Policy{
			Version: 1,
			Bindings: []*iampb.Binding{{
				Role:    "roles/run.invoker",
				Members: []string{"user:alice@example.com"},
			}},
		},
	})
	if err != nil || len(pol.Bindings) != 1 {
		t.Fatalf("SetIamPolicy: %v %+v", err, pol)
	}
	got, err := srv.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: res})
	if err != nil || len(got.Bindings) != 1 {
		t.Fatalf("GetIamPolicy: %v %+v", err, got)
	}
	tp, err := srv.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
		Resource:     res,
		Permissions:  []string{"run.services.get", "run.services.delete"},
	})
	if err != nil {
		t.Fatalf("TestIamPermissions: %v", err)
	}
	if len(tp.Permissions) != 0 {
		t.Fatalf("expected no permissions for unrelated caller, got %v", tp.Permissions)
	}
}

func TestRESTGateway(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := runtime.NewServeMux()
	ctx := context.Background()
	if err := runpb.RegisterServicesHandlerServer(ctx, mux, srv); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// POST /v2/{parent}/services?serviceId=web
	body := `{"template":{"containers":[{"image":"img:latest"}]}}`
	req, _ := http.NewRequest("POST", ts.URL+"/v2/"+location+"/services?serviceId=web", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("create service status = %d body=%s", resp.StatusCode, raw)
	}
	var op map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&op); err != nil {
		t.Fatal(err)
	}
	if op["done"] != true {
		t.Fatalf("operation not done: %v", op)
	}

	// GET service.
	resp2, err := http.Get(ts.URL + "/v2/" + svcName)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		raw, _ := io.ReadAll(resp2.Body)
		t.Fatalf("get service status = %d body=%s", resp2.StatusCode, raw)
	}
	var svc map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&svc); err != nil {
		t.Fatal(err)
	}
	if svc["name"] != svcName {
		t.Fatalf("unexpected service: %v", svc)
	}

	// PATCH with updateMask.
	patch := `{"name":"` + svcName + `","labels":{"env":"prod"}}`
	req3, _ := http.NewRequest("PATCH", ts.URL+"/v2/"+svcName+"?updateMask=labels", strings.NewReader(patch))
	req3.Header.Set("Content-Type", "application/json")
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != 200 {
		raw, _ := io.ReadAll(resp3.Body)
		t.Fatalf("patch status = %d body=%s", resp3.StatusCode, raw)
	}

	// DELETE -> LRO.
	req4, _ := http.NewRequest("DELETE", ts.URL+"/v2/"+svcName, nil)
	resp4, err := http.DefaultClient.Do(req4)
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != 200 {
		raw, _ := io.ReadAll(resp4.Body)
		t.Fatalf("delete status = %d body=%s", resp4.StatusCode, raw)
	}
}
