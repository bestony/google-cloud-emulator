package compute

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

	computepb "github.com/bestony/google-cloud-emulator/gen/google/cloud/compute/v1"
)

const (
	project = "demo"
	zone    = "us-central1-a"
	region  = "us-central1"
)

func newInstances(s *State) *instancesServer    { return &instancesServer{state: s} }
func newDisks(s *State) *disksServer            { return &disksServer{state: s} }
func newNetworks(s *State) *networksServer      { return &networksServer{state: s} }
func newFirewall(s *State) *firewallsServer     { return &firewallsServer{state: s} }
func newImages(s *State) *imagesServer          { return &imagesServer{state: s} }
func newZoneOps(s *State) *zoneOperationsServer { return &zoneOperationsServer{state: s} }
func newGlobalOps(s *State) *globalOperationsServer {
	return &globalOperationsServer{state: s}
}

func TestInstanceLifecycle(t *testing.T) {
	st := NewState()
	srv := newInstances(st)
	ctx := context.Background()

	op, err := srv.Insert(ctx, &computepb.InsertInstanceRequest{
		Project: project,
		Zone:    "zones/" + zone,
		InstanceResource: &computepb.Instance{
			Name:        protoString("web-1"),
			MachineType: protoString("zones/" + zone + "/machineTypes/n1-standard-1"),
			Labels:      map[string]string{"env": "test"},
		},
	})
	if err != nil {
		t.Fatalf("InsertInstance: %v", err)
	}
	if op.GetStatus() != computepb.Operation_DONE {
		t.Fatalf("operation not done: %+v", op)
	}
	if op.GetName() == "" || op.Id == nil {
		t.Fatalf("operation missing name/id: %+v", op)
	}

	// Duplicate -> AlreadyExists.
	if _, err := srv.Insert(ctx, &computepb.InsertInstanceRequest{
		Project:          project,
		Zone:             zone,
		InstanceResource: &computepb.Instance{Name: protoString("web-1")},
	}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate insert: got %v", err)
	}

	inst, err := srv.Get(ctx, &computepb.GetInstanceRequest{Project: project, Zone: zone, Instance: "web-1"})
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if inst.GetName() != "web-1" || inst.GetKind() != "compute#instance" || inst.GetStatus() != "RUNNING" {
		t.Fatalf("unexpected instance: %+v", inst)
	}
	if inst.GetId() == 0 || inst.GetCreationTimestamp() == "" || inst.GetSelfLink() == "" {
		t.Fatalf("instance missing server-set fields: %+v", inst)
	}
	if inst.GetZone() != zoneSelfLink(project, zone) {
		t.Fatalf("instance zone = %q", inst.GetZone())
	}

	// Zone-scoped operation is retrievable by bare id.
	opID := strings.TrimPrefix(op.GetName(), "projects/"+project+"/zones/"+zone+"/operations/")
	gotOp, err := newZoneOps(st).Get(ctx, &computepb.GetZoneOperationRequest{
		Project: project, Zone: zone, Operation: opID,
	})
	if err != nil || gotOp.GetName() != op.GetName() {
		t.Fatalf("GetZoneOperation: %v %+v", err, gotOp)
	}
	zops, err := newZoneOps(st).List(ctx, &computepb.ListZoneOperationsRequest{Project: project, Zone: zone})
	if err != nil || len(zops.Items) != 1 {
		t.Fatalf("ListZoneOperations: %v %+v", err, zops)
	}

	// Aggregated list groups by zone and filters by project.
	agg, err := srv.AggregatedList(ctx, &computepb.AggregatedListInstancesRequest{Project: project})
	if err != nil {
		t.Fatalf("AggregatedListInstances: %v", err)
	}
	scoped, ok := agg.Items["zones/"+zone]
	if !ok || len(scoped.Instances) != 1 || scoped.Instances[0].GetName() != "web-1" {
		t.Fatalf("unexpected aggregated items: %+v", agg.Items)
	}
	aggOther, err := srv.AggregatedList(ctx, &computepb.AggregatedListInstancesRequest{Project: "other"})
	if err != nil || len(aggOther.Items) != 0 {
		t.Fatalf("aggregated list leaked across projects: %+v", aggOther.Items)
	}

	// List + pagination.
	for i := 0; i < 2; i++ {
		if _, err := srv.Insert(ctx, &computepb.InsertInstanceRequest{
			Project:          project,
			Zone:             zone,
			InstanceResource: &computepb.Instance{Name: protoString("web-" + string(rune('2'+i)))},
		}); err != nil {
			t.Fatal(err)
		}
	}
	page1, err := srv.List(ctx, &computepb.ListInstancesRequest{Project: project, Zone: zone, MaxResults: protoUint32(2)})
	if err != nil || len(page1.Items) != 2 || page1.NextPageToken == nil {
		t.Fatalf("ListInstances page1: %v %+v", err, page1)
	}
	page2, err := srv.List(ctx, &computepb.ListInstancesRequest{Project: project, Zone: zone, MaxResults: protoUint32(2), PageToken: nextTokenPtr(page1.GetNextPageToken())})
	if err != nil || len(page2.Items) != 1 {
		t.Fatalf("ListInstances page2: %v %+v", err, page2)
	}

	// Delete -> operation, then Get is NotFound.
	if _, err := srv.Delete(ctx, &computepb.DeleteInstanceRequest{Project: project, Zone: zone, Instance: "web-1"}); err != nil {
		t.Fatalf("DeleteInstance: %v", err)
	}
	if _, err := srv.Get(ctx, &computepb.GetInstanceRequest{Project: project, Zone: zone, Instance: "web-1"}); status.Code(err) != codes.NotFound {
		t.Fatalf("get deleted instance: got %v", err)
	}
	if _, err := srv.Delete(ctx, &computepb.DeleteInstanceRequest{Project: project, Zone: zone, Instance: "web-1"}); status.Code(err) != codes.NotFound {
		t.Fatalf("delete missing instance: got %v", err)
	}
}

func TestNetworkFirewallAddress(t *testing.T) {
	st := NewState()
	ctx := context.Background()
	nw, fw, ad := newNetworks(st), newFirewall(st), &addressesServer{state: st}

	if _, err := nw.Insert(ctx, &computepb.InsertNetworkRequest{
		Project:         project,
		NetworkResource: &computepb.Network{Name: protoString("default")},
	}); err != nil {
		t.Fatalf("InsertNetwork: %v", err)
	}
	net, err := nw.Get(ctx, &computepb.GetNetworkRequest{Project: project, Network: "default"})
	if err != nil || !net.GetAutoCreateSubnetworks() {
		t.Fatalf("GetNetwork: %v %+v", err, net)
	}
	// Patch changes description, keeps id/selfLink.
	if _, err := nw.Patch(ctx, &computepb.PatchNetworkRequest{
		Project:         project,
		Network:         "default",
		NetworkResource: &computepb.Network{Description: protoString("patched")},
	}); err != nil {
		t.Fatalf("PatchNetwork: %v", err)
	}
	net2, _ := nw.Get(ctx, &computepb.GetNetworkRequest{Project: project, Network: "default"})
	if net2.GetDescription() != "patched" || net2.GetSelfLink() != net.GetSelfLink() || net2.GetId() != net.GetId() {
		t.Fatalf("PatchNetwork did not preserve identity: %+v", net2)
	}
	if _, err := nw.Insert(ctx, &computepb.InsertNetworkRequest{Project: project, NetworkResource: &computepb.Network{Name: protoString("default")}}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate network: got %v", err)
	}

	// Global operations recorded for network insert + patch.
	gops, err := newGlobalOps(st).List(ctx, &computepb.ListGlobalOperationsRequest{Project: project})
	if err != nil || len(gops.Items) != 2 {
		t.Fatalf("ListGlobalOperations: %v %+v", err, gops)
	}

	if _, err := fw.Insert(ctx, &computepb.InsertFirewallRequest{
		Project:          project,
		FirewallResource: &computepb.Firewall{Name: protoString("allow-ssh"), Network: protoString("default")},
	}); err != nil {
		t.Fatalf("InsertFirewall: %v", err)
	}
	f, err := fw.Get(ctx, &computepb.GetFirewallRequest{Project: project, Firewall: "allow-ssh"})
	if err != nil || f.GetDirection() != "INGRESS" || f.GetPriority() != 1000 {
		t.Fatalf("GetFirewall: %v %+v", err, f)
	}

	if _, err := ad.Insert(ctx, &computepb.InsertAddressRequest{
		Project:         project,
		Region:          region,
		AddressResource: &computepb.Address{Name: protoString("web-ip"), Address: protoString("10.0.0.1")},
	}); err != nil {
		t.Fatalf("InsertAddress: %v", err)
	}
	a, err := ad.Get(ctx, &computepb.GetAddressRequest{Project: project, Region: region, Address: "web-ip"})
	if err != nil || a.GetAddress() != "10.0.0.1" || a.GetStatus() != "RESERVED" {
		t.Fatalf("GetAddress: %v %+v", err, a)
	}
	aggAddr, err := ad.AggregatedList(ctx, &computepb.AggregatedListAddressesRequest{Project: project})
	if err != nil {
		t.Fatalf("AggregatedListAddresses: %v", err)
	}
	if scoped, ok := aggAddr.Items["regions/"+region]; !ok || len(scoped.Addresses) != 1 {
		t.Fatalf("unexpected address aggregation: %+v", aggAddr.Items)
	}
}

func TestDiskSubnetworkImageSnapshotTemplateHealthCheck(t *testing.T) {
	st := NewState()
	ctx := context.Background()

	// Disk lifecycle.
	d := &disksServer{state: st}
	op, err := d.Insert(ctx, &computepb.InsertDiskRequest{
		Project:      project,
		Zone:         zone,
		DiskResource: &computepb.Disk{Name: protoString("data-1"), SizeGb: protoInt64(10)},
	})
	if err != nil || op.GetStatus() != computepb.Operation_DONE {
		t.Fatalf("InsertDisk: %v %+v", err, op)
	}
	disk, err := d.Get(ctx, &computepb.GetDiskRequest{Project: project, Zone: zone, Disk: "data-1"})
	if err != nil || disk.GetStatus() != "READY" || disk.GetKind() != "compute#disk" {
		t.Fatalf("GetDisk: %v %+v", err, disk)
	}

	// Subnetwork (regional).
	sub := &subnetworksServer{state: st}
	if _, err := sub.Insert(ctx, &computepb.InsertSubnetworkRequest{
		Project:            project,
		Region:             region,
		SubnetworkResource: &computepb.Subnetwork{Name: protoString("subnet-1"), IpCidrRange: protoString("10.0.0.0/24")},
	}); err != nil {
		t.Fatalf("InsertSubnetwork: %v", err)
	}
	sgot, err := sub.Get(ctx, &computepb.GetSubnetworkRequest{Project: project, Region: region, Subnetwork: "subnet-1"})
	if err != nil || sgot.GetIpCidrRange() != "10.0.0.0/24" {
		t.Fatalf("GetSubnetwork: %v %+v", err, sgot)
	}

	// Image + GetFromFamily.
	img := &imagesServer{state: st}
	if _, err := img.Insert(ctx, &computepb.InsertImageRequest{
		Project:       project,
		ImageResource: &computepb.Image{Name: protoString("img-1"), Family: protoString("debian-12")},
	}); err != nil {
		t.Fatalf("InsertImage: %v", err)
	}
	fam, err := img.GetFromFamily(ctx, &computepb.GetFromFamilyImageRequest{Project: project, Family: "debian-12"})
	if err != nil || fam.GetName() != "img-1" {
		t.Fatalf("GetFromFamilyImage: %v %+v", err, fam)
	}
	if _, err := img.GetFromFamily(ctx, &computepb.GetFromFamilyImageRequest{Project: project, Family: "nope"}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing family: got %v", err)
	}

	// Snapshot.
	snap := &snapshotsServer{state: st}
	if _, err := snap.Insert(ctx, &computepb.InsertSnapshotRequest{
		Project:          project,
		SnapshotResource: &computepb.Snapshot{Name: protoString("snap-1"), SourceDisk: protoString(disk.GetSelfLink())},
	}); err != nil {
		t.Fatalf("InsertSnapshot: %v", err)
	}
	sg, err := snap.Get(ctx, &computepb.GetSnapshotRequest{Project: project, Snapshot: "snap-1"})
	if err != nil || sg.GetStatus() != "READY" {
		t.Fatalf("GetSnapshot: %v %+v", err, sg)
	}

	// Instance template (global).
	tpl := &instanceTemplatesServer{state: st}
	if _, err := tpl.Insert(ctx, &computepb.InsertInstanceTemplateRequest{
		Project:                  project,
		InstanceTemplateResource: &computepb.InstanceTemplate{Name: protoString("tpl-1")},
	}); err != nil {
		t.Fatalf("InsertInstanceTemplate: %v", err)
	}
	tg, err := tpl.Get(ctx, &computepb.GetInstanceTemplateRequest{Project: project, InstanceTemplate: "tpl-1"})
	if err != nil || tg.GetKind() != "compute#instanceTemplate" {
		t.Fatalf("GetInstanceTemplate: %v %+v", err, tg)
	}

	// Health check (global).
	hc := &healthChecksServer{state: st}
	if _, err := hc.Insert(ctx, &computepb.InsertHealthCheckRequest{
		Project:             project,
		HealthCheckResource: &computepb.HealthCheck{Name: protoString("hc-1"), Type: protoString("TCP")},
	}); err != nil {
		t.Fatalf("InsertHealthCheck: %v", err)
	}
	hg, err := hc.Get(ctx, &computepb.GetHealthCheckRequest{Project: project, HealthCheck: "hc-1"})
	if err != nil || hg.GetType() != "TCP" {
		t.Fatalf("GetHealthCheck: %v %+v", err, hg)
	}
}

func TestReferenceDataEmpty(t *testing.T) {
	st := NewState()
	ctx := context.Background()

	if _, err := (&machineTypesServer{state: st}).Get(ctx, &computepb.GetMachineTypeRequest{Project: project, Zone: zone, MachineType: "n1-standard-1"}); status.Code(err) != codes.NotFound {
		t.Fatalf("GetMachineType: got %v", err)
	}
	mtl, err := (&machineTypesServer{state: st}).List(ctx, &computepb.ListMachineTypesRequest{Project: project, Zone: zone})
	if err != nil || len(mtl.Items) != 0 {
		t.Fatalf("ListMachineTypes: %v %+v", err, mtl)
	}
	zl, err := (&zonesServer{state: st}).List(ctx, &computepb.ListZonesRequest{Project: project})
	if err != nil || len(zl.Items) != 0 {
		t.Fatalf("ListZones: %v %+v", err, zl)
	}
	rl, err := (&regionsServer{state: st}).List(ctx, &computepb.ListRegionsRequest{Project: project})
	if err != nil || len(rl.Items) != 0 {
		t.Fatalf("ListRegions: %v %+v", err, rl)
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

	client := computepb.NewInstancesClient(conn)
	if _, err := client.Insert(context.Background(), &computepb.InsertInstanceRequest{
		Project:          project,
		Zone:             zone,
		InstanceResource: &computepb.Instance{Name: protoString("grpc-1")},
	}); err != nil {
		t.Fatalf("gRPC InsertInstance: %v", err)
	}
	got, err := client.Get(context.Background(), &computepb.GetInstanceRequest{Project: project, Zone: zone, Instance: "grpc-1"})
	if err != nil || got.GetName() != "grpc-1" {
		t.Fatalf("gRPC GetInstance: %v %+v", err, got)
	}
	// Long-tail compute services use the generic stateful control plane.
	machineImages := computepb.NewMachineImagesClient(conn)
	if _, err := machineImages.Get(context.Background(), &computepb.GetMachineImageRequest{Project: project, MachineImage: "x"}); status.Code(err) != codes.NotFound {
		t.Fatalf("MachineImages.GetMachineImage before insert: got %v, want NotFound", err)
	}
	if _, err := machineImages.Insert(context.Background(), &computepb.InsertMachineImageRequest{
		Project:              project,
		MachineImageResource: &computepb.MachineImage{Name: protoString("mi-1")},
	}); err != nil {
		t.Fatalf("MachineImages.Insert: %v", err)
	}
	mi, err := machineImages.Get(context.Background(), &computepb.GetMachineImageRequest{Project: project, MachineImage: "mi-1"})
	if err != nil || mi.GetName() != "mi-1" || mi.GetSelfLink() == "" {
		t.Fatalf("MachineImages.Get: %v %+v", err, mi)
	}
	list, err := machineImages.List(context.Background(), &computepb.ListMachineImagesRequest{Project: project})
	if err != nil || len(list.Items) != 1 || list.Items[0].GetName() != "mi-1" {
		t.Fatalf("MachineImages.List: %v %+v", err, list)
	}
	if _, err := machineImages.Delete(context.Background(), &computepb.DeleteMachineImageRequest{Project: project, MachineImage: "mi-1"}); err != nil {
		t.Fatalf("MachineImages.Delete: %v", err)
	}
	if _, err := machineImages.Get(context.Background(), &computepb.GetMachineImageRequest{Project: project, MachineImage: "mi-1"}); status.Code(err) != codes.NotFound {
		t.Fatalf("MachineImages.Get after delete: got %v, want NotFound", err)
	}
}

func TestRESTGateway(t *testing.T) {
	st := NewState()
	mux := runtime.NewServeMux()
	if err := Register(grpc.NewServer(), mux, st); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Insert instance via REST.
	body := `{"name":"rest-1","machineType":"zones/us-central1-a/machineTypes/n1-standard-1"}`
	req, _ := http.NewRequest("POST", ts.URL+"/compute/v1/projects/"+project+"/zones/"+zone+"/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("insert instance status = %d body=%s", resp.StatusCode, raw)
	}
	var op map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&op); err != nil {
		t.Fatal(err)
	}
	if op["status"] != "DONE" || op["name"] == "" {
		t.Fatalf("unexpected operation: %v", op)
	}

	// Get instance.
	resp2, err := http.Get(ts.URL + "/compute/v1/projects/" + project + "/zones/" + zone + "/instances/rest-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var inst map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&inst); err != nil {
		t.Fatal(err)
	}
	if inst["name"] != "rest-1" || inst["status"] != "RUNNING" {
		t.Fatalf("unexpected instance: %v", inst)
	}

	// Delete -> 200 with operation.
	req3, _ := http.NewRequest("DELETE", ts.URL+"/compute/v1/projects/"+project+"/zones/"+zone+"/instances/rest-1", nil)
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != 200 {
		raw, _ := io.ReadAll(resp3.Body)
		t.Fatalf("delete status = %d body=%s", resp3.StatusCode, raw)
	}

	// List networks is empty but well-formed.
	resp4, err := http.Get(ts.URL + "/compute/v1/projects/" + project + "/global/networks")
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	var nl map[string]any
	if err := json.NewDecoder(resp4.Body).Decode(&nl); err != nil {
		t.Fatal(err)
	}
	if nl["kind"] != "compute#networkList" {
		t.Fatalf("unexpected network list: %v", nl)
	}

	// A long-tail resource is served through the same generated REST gateway.
	miBody := `{"name":"rest-mi"}`
	miReq, _ := http.NewRequest("POST", ts.URL+"/compute/v1/projects/"+project+"/global/machineImages", strings.NewReader(miBody))
	miReq.Header.Set("Content-Type", "application/json")
	miResp, err := http.DefaultClient.Do(miReq)
	if err != nil {
		t.Fatal(err)
	}
	defer miResp.Body.Close()
	if miResp.StatusCode != 200 {
		raw, _ := io.ReadAll(miResp.Body)
		t.Fatalf("insert machine image status = %d body=%s", miResp.StatusCode, raw)
	}
	miGet, err := http.Get(ts.URL + "/compute/v1/projects/" + project + "/global/machineImages/rest-mi")
	if err != nil {
		t.Fatal(err)
	}
	defer miGet.Body.Close()
	var mi map[string]any
	if err := json.NewDecoder(miGet.Body).Decode(&mi); err != nil {
		t.Fatal(err)
	}
	if mi["name"] != "rest-mi" || mi["kind"] != "compute#machineImage" {
		t.Fatalf("unexpected machine image: %v", mi)
	}
}
