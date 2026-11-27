package compute

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	computepb "github.com/bestony/google-cloud-emulator/gen/google/cloud/compute/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Each service gets its own server type embedding only its own
// Unimplemented server, avoiding ambiguous selectors on shared method names.
type instancesServer struct {
	computepb.UnimplementedInstancesServer
	state *State
}

type disksServer struct {
	computepb.UnimplementedDisksServer
	state *State
}

type networksServer struct {
	computepb.UnimplementedNetworksServer
	state *State
}

type subnetworksServer struct {
	computepb.UnimplementedSubnetworksServer
	state *State
}

type firewallsServer struct {
	computepb.UnimplementedFirewallsServer
	state *State
}

type addressesServer struct {
	computepb.UnimplementedAddressesServer
	state *State
}

type imagesServer struct {
	computepb.UnimplementedImagesServer
	state *State
}

type snapshotsServer struct {
	computepb.UnimplementedSnapshotsServer
	state *State
}

type instanceTemplatesServer struct {
	computepb.UnimplementedInstanceTemplatesServer
	state *State
}

type healthChecksServer struct {
	computepb.UnimplementedHealthChecksServer
	state *State
}

type machineTypesServer struct {
	computepb.UnimplementedMachineTypesServer
	state *State
}

type zonesServer struct {
	computepb.UnimplementedZonesServer
	state *State
}

type regionsServer struct {
	computepb.UnimplementedRegionsServer
	state *State
}

type globalOperationsServer struct {
	computepb.UnimplementedGlobalOperationsServer
	state *State
}

type zoneOperationsServer struct {
	computepb.UnimplementedZoneOperationsServer
	state *State
}

type regionOperationsServer struct {
	computepb.UnimplementedRegionOperationsServer
	state *State
}

func notFound(what, project, name string) error {
	return status.Errorf(codes.NotFound, "The %s '%s' does not exist in project '%s'", what, name, project)
}

func alreadyExists(what, project, name string) error {
	return status.Errorf(codes.AlreadyExists, "The %s '%s' already exists in project '%s'", what, name, project)
}

func invalidArg(format string, args ...any) error {
	return status.Errorf(codes.InvalidArgument, format, args...)
}

// paginate applies compute-style maxResults/pageToken (offset encoded as a
// decimal string) over an already-sorted slice.
func paginate[T any](items []T, maxResults *uint32, pageToken *string) ([]T, string) {
	start := 0
	if pageToken != nil && *pageToken != "" {
		if n, err := strconv.Atoi(*pageToken); err == nil && n > 0 {
			start = n
		}
	}
	if start > len(items) {
		start = len(items)
	}
	if maxResults == nil || *maxResults == 0 || int(*maxResults) >= len(items)-start {
		return items[start:], ""
	}
	end := start + int(*maxResults)
	return items[start:end], strconv.Itoa(end)
}

func byName[T interface{ GetName() string }](items []T) {
	sort.Slice(items, func(i, j int) bool { return items[i].GetName() < items[j].GetName() })
}

func nextTokenPtr(next string) *string {
	if next == "" {
		return nil
	}
	return protoString(next)
}

// ---------------------------------------------------------------------------
// Instances
// ---------------------------------------------------------------------------

func instanceKey(project, zone, name string) string {
	return resourceKey(project, zoneName(zone), name)
}

func (s *instancesServer) Get(_ context.Context, req *computepb.GetInstanceRequest) (*computepb.Instance, error) {
	if req.Project == "" || req.Zone == "" || req.Instance == "" {
		return nil, invalidArg("project, zone and instance are required")
	}
	s.state.mu.RLock()
	inst, ok := s.state.instances[instanceKey(req.Project, req.Zone, req.Instance)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("instance", req.Project, req.Instance)
	}
	return proto.Clone(inst).(*computepb.Instance), nil
}

func (s *instancesServer) Insert(_ context.Context, req *computepb.InsertInstanceRequest) (*computepb.Operation, error) {
	inst := req.InstanceResource
	if inst == nil || derefString(inst.Name) == "" {
		return nil, invalidArg("instance resource with a name is required")
	}
	zone := zoneName(req.Zone)
	if zone == "" {
		return nil, invalidArg("zone is required")
	}
	name := derefString(inst.Name)
	key := instanceKey(req.Project, zone, name)

	s.state.mu.Lock()
	if _, exists := s.state.instances[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("instance", req.Project, name)
	}
	inst = proto.Clone(inst).(*computepb.Instance)
	inst.Id = protoUint64(s.state.nextID())
	inst.Kind = protoString("compute#instance")
	inst.Name = protoString(name)
	inst.CreationTimestamp = protoString(rfc3339Now())
	inst.SelfLink = protoString(fmt.Sprintf("%s/projects/%s/zones/%s/instances/%s", selfLinkBase, req.Project, zone, name))
	inst.Zone = protoString(zoneSelfLink(req.Project, zone))
	if inst.Status == nil {
		inst.Status = protoString("RUNNING")
	}
	if inst.Labels == nil {
		inst.Labels = map[string]string{}
	}
	s.state.instances[key] = inst
	target := inst.GetSelfLink()
	op := s.state.addOperation("projects/"+req.Project+"/zones/"+zone, "insert", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *instancesServer) Delete(_ context.Context, req *computepb.DeleteInstanceRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Zone == "" || req.Instance == "" {
		return nil, invalidArg("project, zone and instance are required")
	}
	zone := zoneName(req.Zone)
	key := instanceKey(req.Project, zone, req.Instance)
	s.state.mu.Lock()
	inst, ok := s.state.instances[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("instance", req.Project, req.Instance)
	}
	target := inst.GetSelfLink()
	delete(s.state.instances, key)
	op := s.state.addOperation("projects/"+req.Project+"/zones/"+zone, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *instancesServer) List(_ context.Context, req *computepb.ListInstancesRequest) (*computepb.InstanceList, error) {
	if req.Project == "" || req.Zone == "" {
		return nil, invalidArg("project and zone are required")
	}
	zone := zoneName(req.Zone)
	s.state.mu.RLock()
	var items []*computepb.Instance
	prefix := resourceKey(req.Project, zone, "")
	for key, inst := range s.state.instances {
		if strings.HasPrefix(key, prefix) {
			items = append(items, inst)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.InstanceList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/zones/%s/instances", selfLinkBase, req.Project, zone)),
		Kind:          protoString("compute#instanceList"),
	}, nil
}

func (s *instancesServer) AggregatedList(_ context.Context, req *computepb.AggregatedListInstancesRequest) (*computepb.InstanceAggregatedList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	groups := map[string][]*computepb.Instance{}
	for key, inst := range s.state.instances {
		parts := strings.Split(key, "|")
		if len(parts) < 3 || parts[0] != req.Project {
			continue
		}
		groups[parts[1]] = append(groups[parts[1]], inst)
	}
	s.state.mu.RUnlock()

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := &computepb.InstanceAggregatedList{
		Id:    protoString("compute.instances.aggregatedList"),
		Kind:  protoString("compute#instanceAggregatedList"),
		Items: map[string]*computepb.InstancesScopedList{},
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/aggregated/instances",
			selfLinkBase, req.Project)),
	}
	for _, zn := range keys {
		items := groups[zn]
		byName(items)
		out.Items["zones/"+zn] = &computepb.InstancesScopedList{
			Instances: items,
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Disks
// ---------------------------------------------------------------------------

func diskKey(project, zone, name string) string {
	return resourceKey(project, zoneName(zone), name)
}

func (s *disksServer) Get(_ context.Context, req *computepb.GetDiskRequest) (*computepb.Disk, error) {
	if req.Project == "" || req.Zone == "" || req.Disk == "" {
		return nil, invalidArg("project, zone and disk are required")
	}
	s.state.mu.RLock()
	d, ok := s.state.disks[diskKey(req.Project, req.Zone, req.Disk)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("disk", req.Project, req.Disk)
	}
	return proto.Clone(d).(*computepb.Disk), nil
}

func (s *disksServer) Insert(_ context.Context, req *computepb.InsertDiskRequest) (*computepb.Operation, error) {
	d := req.DiskResource
	if d == nil || derefString(d.Name) == "" {
		return nil, invalidArg("disk resource with a name is required")
	}
	zone := zoneName(req.Zone)
	if zone == "" {
		return nil, invalidArg("zone is required")
	}
	name := derefString(d.Name)
	key := diskKey(req.Project, zone, name)

	s.state.mu.Lock()
	if _, exists := s.state.disks[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("disk", req.Project, name)
	}
	d = proto.Clone(d).(*computepb.Disk)
	d.Id = protoUint64(s.state.nextID())
	d.Kind = protoString("compute#disk")
	d.Name = protoString(name)
	d.CreationTimestamp = protoString(rfc3339Now())
	d.SelfLink = protoString(fmt.Sprintf("%s/projects/%s/zones/%s/disks/%s", selfLinkBase, req.Project, zone, name))
	d.Zone = protoString(zoneSelfLink(req.Project, zone))
	if d.Status == nil {
		d.Status = protoString("READY")
	}
	s.state.disks[key] = d
	op := s.state.addOperation("projects/"+req.Project+"/zones/"+zone, "insert", d.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *disksServer) Delete(_ context.Context, req *computepb.DeleteDiskRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Zone == "" || req.Disk == "" {
		return nil, invalidArg("project, zone and disk are required")
	}
	zone := zoneName(req.Zone)
	key := diskKey(req.Project, zone, req.Disk)
	s.state.mu.Lock()
	d, ok := s.state.disks[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("disk", req.Project, req.Disk)
	}
	target := d.GetSelfLink()
	delete(s.state.disks, key)
	op := s.state.addOperation("projects/"+req.Project+"/zones/"+zone, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *disksServer) List(_ context.Context, req *computepb.ListDisksRequest) (*computepb.DiskList, error) {
	if req.Project == "" || req.Zone == "" {
		return nil, invalidArg("project and zone are required")
	}
	zone := zoneName(req.Zone)
	s.state.mu.RLock()
	var items []*computepb.Disk
	prefix := resourceKey(req.Project, zone, "")
	for key, d := range s.state.disks {
		if strings.HasPrefix(key, prefix) {
			items = append(items, d)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.DiskList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/zones/%s/disks", selfLinkBase, req.Project, zone)),
		Kind:          protoString("compute#diskList"),
	}, nil
}

func (s *disksServer) AggregatedList(_ context.Context, req *computepb.AggregatedListDisksRequest) (*computepb.DiskAggregatedList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	groups := map[string][]*computepb.Disk{}
	for key, d := range s.state.disks {
		parts := strings.Split(key, "|")
		if len(parts) < 3 || parts[0] != req.Project {
			continue
		}
		groups[parts[1]] = append(groups[parts[1]], d)
	}
	s.state.mu.RUnlock()

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := &computepb.DiskAggregatedList{
		Id:    protoString("compute.disks.aggregatedList"),
		Kind:  protoString("compute#diskAggregatedList"),
		Items: map[string]*computepb.DisksScopedList{},
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/aggregated/disks",
			selfLinkBase, req.Project)),
	}
	for _, zn := range keys {
		items := groups[zn]
		byName(items)
		out.Items["zones/"+zn] = &computepb.DisksScopedList{
			Disks: items,
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Networks
// ---------------------------------------------------------------------------

func (s *networksServer) Get(_ context.Context, req *computepb.GetNetworkRequest) (*computepb.Network, error) {
	if req.Project == "" || req.Network == "" {
		return nil, invalidArg("project and network are required")
	}
	s.state.mu.RLock()
	n, ok := s.state.networks[resourceKey(req.Project, req.Network)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("network", req.Project, req.Network)
	}
	return proto.Clone(n).(*computepb.Network), nil
}

func (s *networksServer) Insert(_ context.Context, req *computepb.InsertNetworkRequest) (*computepb.Operation, error) {
	n := req.NetworkResource
	if n == nil || derefString(n.Name) == "" {
		return nil, invalidArg("network resource with a name is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := derefString(n.Name)
	key := resourceKey(req.Project, name)

	s.state.mu.Lock()
	if _, exists := s.state.networks[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("network", req.Project, name)
	}
	n = proto.Clone(n).(*computepb.Network)
	n.Id = protoUint64(s.state.nextID())
	n.Kind = protoString("compute#network")
	n.Name = protoString(name)
	n.CreationTimestamp = protoString(rfc3339Now())
	n.SelfLink = protoString(globalSelfLink(req.Project, "networks", name))
	if n.AutoCreateSubnetworks == nil {
		n.AutoCreateSubnetworks = protoBool(true)
	}
	s.state.networks[key] = n
	op := s.state.addOperation("projects/"+req.Project, "insert", n.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *networksServer) Delete(_ context.Context, req *computepb.DeleteNetworkRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Network == "" {
		return nil, invalidArg("project and network are required")
	}
	key := resourceKey(req.Project, req.Network)
	s.state.mu.Lock()
	n, ok := s.state.networks[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("network", req.Project, req.Network)
	}
	target := n.GetSelfLink()
	delete(s.state.networks, key)
	op := s.state.addOperation("projects/"+req.Project, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *networksServer) List(_ context.Context, req *computepb.ListNetworksRequest) (*computepb.NetworkList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	var items []*computepb.Network
	prefix := req.Project + "|"
	for key, n := range s.state.networks {
		if strings.HasPrefix(key, prefix) {
			items = append(items, n)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.NetworkList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/networks", selfLinkBase, req.Project)),
		Kind:          protoString("compute#networkList"),
	}, nil
}

func (s *networksServer) Patch(_ context.Context, req *computepb.PatchNetworkRequest) (*computepb.Operation, error) {
	if req.NetworkResource == nil {
		return nil, invalidArg("network resource is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := req.GetNetwork()
	if name == "" {
		name = derefString(req.NetworkResource.Name)
	}
	if name == "" {
		return nil, invalidArg("network name is required")
	}
	key := resourceKey(req.Project, name)
	s.state.mu.Lock()
	stored, ok := s.state.networks[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("network", req.Project, name)
	}
	patched := proto.Clone(req.NetworkResource).(*computepb.Network)
	patched.Id = stored.Id
	patched.CreationTimestamp = stored.CreationTimestamp
	patched.SelfLink = stored.SelfLink
	patched.Kind = stored.Kind
	patched.Name = stored.Name
	if patched.AutoCreateSubnetworks == nil {
		patched.AutoCreateSubnetworks = stored.AutoCreateSubnetworks
	}
	s.state.networks[key] = patched
	op := s.state.addOperation("projects/"+req.Project, "update", patched.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

// ---------------------------------------------------------------------------
// Subnetworks
// ---------------------------------------------------------------------------

func subnetworkKey(project, region, name string) string {
	return resourceKey(project, regionName(region), name)
}

func subnetworkSelfLink(project, region, name string) string {
	return fmt.Sprintf("%s/projects/%s/regions/%s/subnetworks/%s", selfLinkBase, project, region, name)
}

func (s *subnetworksServer) Get(_ context.Context, req *computepb.GetSubnetworkRequest) (*computepb.Subnetwork, error) {
	if req.Project == "" || req.Region == "" || req.Subnetwork == "" {
		return nil, invalidArg("project, region and subnetwork are required")
	}
	s.state.mu.RLock()
	sub, ok := s.state.subnetworks[subnetworkKey(req.Project, req.Region, req.Subnetwork)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("subnetwork", req.Project, req.Subnetwork)
	}
	return proto.Clone(sub).(*computepb.Subnetwork), nil
}

func (s *subnetworksServer) Insert(_ context.Context, req *computepb.InsertSubnetworkRequest) (*computepb.Operation, error) {
	sub := req.SubnetworkResource
	if sub == nil || derefString(sub.Name) == "" {
		return nil, invalidArg("subnetwork resource with a name is required")
	}
	region := regionName(req.Region)
	if region == "" || req.Project == "" {
		return nil, invalidArg("project and region are required")
	}
	name := derefString(sub.Name)
	key := subnetworkKey(req.Project, region, name)

	s.state.mu.Lock()
	if _, exists := s.state.subnetworks[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("subnetwork", req.Project, name)
	}
	sub = proto.Clone(sub).(*computepb.Subnetwork)
	sub.Id = protoUint64(s.state.nextID())
	sub.Kind = protoString("compute#subnetwork")
	sub.Name = protoString(name)
	sub.CreationTimestamp = protoString(rfc3339Now())
	sub.SelfLink = protoString(subnetworkSelfLink(req.Project, region, name))
	sub.Region = protoString(regionSelfLink(req.Project, region))
	s.state.subnetworks[key] = sub
	op := s.state.addOperation("projects/"+req.Project+"/regions/"+region, "insert", sub.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *subnetworksServer) Delete(_ context.Context, req *computepb.DeleteSubnetworkRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Region == "" || req.Subnetwork == "" {
		return nil, invalidArg("project, region and subnetwork are required")
	}
	region := regionName(req.Region)
	key := subnetworkKey(req.Project, region, req.Subnetwork)
	s.state.mu.Lock()
	sub, ok := s.state.subnetworks[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("subnetwork", req.Project, req.Subnetwork)
	}
	target := sub.GetSelfLink()
	delete(s.state.subnetworks, key)
	op := s.state.addOperation("projects/"+req.Project+"/regions/"+region, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *subnetworksServer) List(_ context.Context, req *computepb.ListSubnetworksRequest) (*computepb.SubnetworkList, error) {
	if req.Project == "" || req.Region == "" {
		return nil, invalidArg("project and region are required")
	}
	region := regionName(req.Region)
	s.state.mu.RLock()
	var items []*computepb.Subnetwork
	prefix := resourceKey(req.Project, region, "")
	for key, sub := range s.state.subnetworks {
		if strings.HasPrefix(key, prefix) {
			items = append(items, sub)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.SubnetworkList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/regions/%s/subnetworks", selfLinkBase, req.Project, region)),
		Kind:          protoString("compute#subnetworkList"),
	}, nil
}

func (s *subnetworksServer) Patch(_ context.Context, req *computepb.PatchSubnetworkRequest) (*computepb.Operation, error) {
	if req.SubnetworkResource == nil {
		return nil, invalidArg("subnetwork resource is required")
	}
	if req.Project == "" || req.Region == "" {
		return nil, invalidArg("project and region are required")
	}
	region := regionName(req.Region)
	name := req.GetSubnetwork()
	if name == "" {
		name = derefString(req.SubnetworkResource.Name)
	}
	if name == "" {
		return nil, invalidArg("subnetwork name is required")
	}
	key := subnetworkKey(req.Project, region, name)
	s.state.mu.Lock()
	stored, ok := s.state.subnetworks[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("subnetwork", req.Project, name)
	}
	patched := proto.Clone(req.SubnetworkResource).(*computepb.Subnetwork)
	patched.Id = stored.Id
	patched.CreationTimestamp = stored.CreationTimestamp
	patched.SelfLink = stored.SelfLink
	patched.Kind = stored.Kind
	patched.Name = stored.Name
	patched.Region = stored.Region
	s.state.subnetworks[key] = patched
	op := s.state.addOperation("projects/"+req.Project+"/regions/"+region, "update", patched.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *subnetworksServer) AggregatedList(_ context.Context, req *computepb.AggregatedListSubnetworksRequest) (*computepb.SubnetworkAggregatedList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	groups := map[string][]*computepb.Subnetwork{}
	for key, sub := range s.state.subnetworks {
		parts := strings.Split(key, "|")
		if len(parts) < 3 || parts[0] != req.Project {
			continue
		}
		groups[parts[1]] = append(groups[parts[1]], sub)
	}
	s.state.mu.RUnlock()

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := &computepb.SubnetworkAggregatedList{
		Id:    protoString("compute.subnetworks.aggregatedList"),
		Kind:  protoString("compute#subnetworkAggregatedList"),
		Items: map[string]*computepb.SubnetworksScopedList{},
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/aggregated/subnetworks",
			selfLinkBase, req.Project)),
	}
	for _, rn := range keys {
		items := groups[rn]
		byName(items)
		out.Items["regions/"+rn] = &computepb.SubnetworksScopedList{
			Subnetworks: items,
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Firewalls
// ---------------------------------------------------------------------------

func (s *firewallsServer) Get(_ context.Context, req *computepb.GetFirewallRequest) (*computepb.Firewall, error) {
	if req.Project == "" || req.Firewall == "" {
		return nil, invalidArg("project and firewall are required")
	}
	s.state.mu.RLock()
	f, ok := s.state.firewalls[resourceKey(req.Project, req.Firewall)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("firewall", req.Project, req.Firewall)
	}
	return proto.Clone(f).(*computepb.Firewall), nil
}

func (s *firewallsServer) Insert(_ context.Context, req *computepb.InsertFirewallRequest) (*computepb.Operation, error) {
	f := req.FirewallResource
	if f == nil || derefString(f.Name) == "" {
		return nil, invalidArg("firewall resource with a name is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := derefString(f.Name)
	key := resourceKey(req.Project, name)

	s.state.mu.Lock()
	if _, exists := s.state.firewalls[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("firewall", req.Project, name)
	}
	f = proto.Clone(f).(*computepb.Firewall)
	f.Id = protoUint64(s.state.nextID())
	f.Kind = protoString("compute#firewall")
	f.Name = protoString(name)
	f.CreationTimestamp = protoString(rfc3339Now())
	f.SelfLink = protoString(globalSelfLink(req.Project, "firewalls", name))
	if f.Direction == nil {
		f.Direction = protoString("INGRESS")
	}
	if f.Priority == nil {
		f.Priority = protoInt32(1000)
	}
	s.state.firewalls[key] = f
	op := s.state.addOperation("projects/"+req.Project, "insert", f.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *firewallsServer) Delete(_ context.Context, req *computepb.DeleteFirewallRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Firewall == "" {
		return nil, invalidArg("project and firewall are required")
	}
	key := resourceKey(req.Project, req.Firewall)
	s.state.mu.Lock()
	f, ok := s.state.firewalls[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("firewall", req.Project, req.Firewall)
	}
	target := f.GetSelfLink()
	delete(s.state.firewalls, key)
	op := s.state.addOperation("projects/"+req.Project, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *firewallsServer) List(_ context.Context, req *computepb.ListFirewallsRequest) (*computepb.FirewallList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	var items []*computepb.Firewall
	prefix := req.Project + "|"
	for key, f := range s.state.firewalls {
		if strings.HasPrefix(key, prefix) {
			items = append(items, f)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.FirewallList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/firewalls", selfLinkBase, req.Project)),
		Kind:          protoString("compute#firewallList"),
	}, nil
}

func (s *firewallsServer) Patch(_ context.Context, req *computepb.PatchFirewallRequest) (*computepb.Operation, error) {
	if req.FirewallResource == nil {
		return nil, invalidArg("firewall resource is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := req.GetFirewall()
	if name == "" {
		name = derefString(req.FirewallResource.Name)
	}
	if name == "" {
		return nil, invalidArg("firewall name is required")
	}
	key := resourceKey(req.Project, name)
	s.state.mu.Lock()
	stored, ok := s.state.firewalls[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("firewall", req.Project, name)
	}
	patched := proto.Clone(req.FirewallResource).(*computepb.Firewall)
	patched.Id = stored.Id
	patched.CreationTimestamp = stored.CreationTimestamp
	patched.SelfLink = stored.SelfLink
	patched.Kind = stored.Kind
	patched.Name = stored.Name
	s.state.firewalls[key] = patched
	op := s.state.addOperation("projects/"+req.Project, "update", patched.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

// ---------------------------------------------------------------------------
// Addresses (regional)
// ---------------------------------------------------------------------------

func addressKey(project, region, name string) string {
	return resourceKey(project, regionName(region), name)
}

func addressSelfLink(project, region, name string) string {
	return fmt.Sprintf("%s/projects/%s/regions/%s/addresses/%s", selfLinkBase, project, region, name)
}

func (s *addressesServer) Get(_ context.Context, req *computepb.GetAddressRequest) (*computepb.Address, error) {
	if req.Project == "" || req.Region == "" || req.Address == "" {
		return nil, invalidArg("project, region and address are required")
	}
	s.state.mu.RLock()
	a, ok := s.state.addresses[addressKey(req.Project, req.Region, req.Address)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("address", req.Project, req.Address)
	}
	return proto.Clone(a).(*computepb.Address), nil
}

func (s *addressesServer) Insert(_ context.Context, req *computepb.InsertAddressRequest) (*computepb.Operation, error) {
	a := req.AddressResource
	if a == nil || derefString(a.Name) == "" {
		return nil, invalidArg("address resource with a name is required")
	}
	region := regionName(req.Region)
	if region == "" || req.Project == "" {
		return nil, invalidArg("project and region are required")
	}
	name := derefString(a.Name)
	key := addressKey(req.Project, region, name)

	s.state.mu.Lock()
	if _, exists := s.state.addresses[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("address", req.Project, name)
	}
	a = proto.Clone(a).(*computepb.Address)
	a.Id = protoUint64(s.state.nextID())
	a.Kind = protoString("compute#address")
	a.Name = protoString(name)
	a.CreationTimestamp = protoString(rfc3339Now())
	a.SelfLink = protoString(addressSelfLink(req.Project, region, name))
	a.Region = protoString(regionSelfLink(req.Project, region))
	if a.Status == nil {
		a.Status = protoString("RESERVED")
	}
	if a.Address == nil {
		// Emulator does not allocate real IPs; leave the address unset rather
		// than fabricate one.
	}
	s.state.addresses[key] = a
	op := s.state.addOperation("projects/"+req.Project+"/regions/"+region, "insert", a.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *addressesServer) Delete(_ context.Context, req *computepb.DeleteAddressRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Region == "" || req.Address == "" {
		return nil, invalidArg("project, region and address are required")
	}
	region := regionName(req.Region)
	key := addressKey(req.Project, region, req.Address)
	s.state.mu.Lock()
	a, ok := s.state.addresses[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("address", req.Project, req.Address)
	}
	target := a.GetSelfLink()
	delete(s.state.addresses, key)
	op := s.state.addOperation("projects/"+req.Project+"/regions/"+region, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *addressesServer) List(_ context.Context, req *computepb.ListAddressesRequest) (*computepb.AddressList, error) {
	if req.Project == "" || req.Region == "" {
		return nil, invalidArg("project and region are required")
	}
	region := regionName(req.Region)
	s.state.mu.RLock()
	var items []*computepb.Address
	prefix := resourceKey(req.Project, region, "")
	for key, a := range s.state.addresses {
		if strings.HasPrefix(key, prefix) {
			items = append(items, a)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.AddressList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/regions/%s/addresses", selfLinkBase, req.Project, region)),
		Kind:          protoString("compute#addressList"),
	}, nil
}

func (s *addressesServer) AggregatedList(_ context.Context, req *computepb.AggregatedListAddressesRequest) (*computepb.AddressAggregatedList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	groups := map[string][]*computepb.Address{}
	for key, a := range s.state.addresses {
		parts := strings.Split(key, "|")
		if len(parts) < 3 || parts[0] != req.Project {
			continue
		}
		groups[parts[1]] = append(groups[parts[1]], a)
	}
	s.state.mu.RUnlock()

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := &computepb.AddressAggregatedList{
		Id:    protoString("compute.addresses.aggregatedList"),
		Kind:  protoString("compute#addressAggregatedList"),
		Items: map[string]*computepb.AddressesScopedList{},
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/aggregated/addresses",
			selfLinkBase, req.Project)),
	}
	for _, rn := range keys {
		items := groups[rn]
		byName(items)
		out.Items["regions/"+rn] = &computepb.AddressesScopedList{
			Addresses: items,
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Images (global)
// ---------------------------------------------------------------------------

func (s *imagesServer) Get(_ context.Context, req *computepb.GetImageRequest) (*computepb.Image, error) {
	if req.Project == "" || req.Image == "" {
		return nil, invalidArg("project and image are required")
	}
	s.state.mu.RLock()
	img, ok := s.state.images[resourceKey(req.Project, req.Image)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("image", req.Project, req.Image)
	}
	return proto.Clone(img).(*computepb.Image), nil
}

func (s *imagesServer) GetFromFamily(_ context.Context, req *computepb.GetFromFamilyImageRequest) (*computepb.Image, error) {
	if req.Project == "" || req.Family == "" {
		return nil, invalidArg("project and family are required")
	}
	s.state.mu.RLock()
	var matches []*computepb.Image
	for key, img := range s.state.images {
		if strings.HasPrefix(key, req.Project+"|") && img.GetFamily() == req.Family {
			matches = append(matches, img)
		}
	}
	s.state.mu.RUnlock()
	if len(matches) == 0 {
		return nil, notFound("image family", req.Project, req.Family)
	}
	byName(matches)
	return proto.Clone(matches[0]).(*computepb.Image), nil
}

func (s *imagesServer) Insert(_ context.Context, req *computepb.InsertImageRequest) (*computepb.Operation, error) {
	img := req.ImageResource
	if img == nil || derefString(img.Name) == "" {
		return nil, invalidArg("image resource with a name is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	force := req.ForceCreate != nil && req.GetForceCreate()
	name := derefString(img.Name)
	key := resourceKey(req.Project, name)

	s.state.mu.Lock()
	if _, exists := s.state.images[key]; exists && !force {
		s.state.mu.Unlock()
		return nil, alreadyExists("image", req.Project, name)
	}
	img = proto.Clone(img).(*computepb.Image)
	img.Id = protoUint64(s.state.nextID())
	img.Kind = protoString("compute#image")
	img.Name = protoString(name)
	img.CreationTimestamp = protoString(rfc3339Now())
	img.SelfLink = protoString(globalSelfLink(req.Project, "images", name))
	if img.Status == nil {
		img.Status = protoString("READY")
	}
	s.state.images[key] = img
	op := s.state.addOperation("projects/"+req.Project, "insert", img.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *imagesServer) Delete(_ context.Context, req *computepb.DeleteImageRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Image == "" {
		return nil, invalidArg("project and image are required")
	}
	key := resourceKey(req.Project, req.Image)
	s.state.mu.Lock()
	img, ok := s.state.images[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("image", req.Project, req.Image)
	}
	target := img.GetSelfLink()
	delete(s.state.images, key)
	op := s.state.addOperation("projects/"+req.Project, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *imagesServer) List(_ context.Context, req *computepb.ListImagesRequest) (*computepb.ImageList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	var items []*computepb.Image
	prefix := req.Project + "|"
	for key, img := range s.state.images {
		if strings.HasPrefix(key, prefix) {
			items = append(items, img)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.ImageList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/images", selfLinkBase, req.Project)),
		Kind:          protoString("compute#imageList"),
	}, nil
}

func (s *imagesServer) Patch(_ context.Context, req *computepb.PatchImageRequest) (*computepb.Operation, error) {
	if req.ImageResource == nil {
		return nil, invalidArg("image resource is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := req.GetImage()
	if name == "" {
		name = derefString(req.ImageResource.Name)
	}
	if name == "" {
		return nil, invalidArg("image name is required")
	}
	key := resourceKey(req.Project, name)
	s.state.mu.Lock()
	stored, ok := s.state.images[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("image", req.Project, name)
	}
	patched := proto.Clone(req.ImageResource).(*computepb.Image)
	patched.Id = stored.Id
	patched.CreationTimestamp = stored.CreationTimestamp
	patched.SelfLink = stored.SelfLink
	patched.Kind = stored.Kind
	patched.Name = stored.Name
	s.state.images[key] = patched
	op := s.state.addOperation("projects/"+req.Project, "update", patched.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

// ---------------------------------------------------------------------------
// Snapshots (global)
// ---------------------------------------------------------------------------

func (s *snapshotsServer) Get(_ context.Context, req *computepb.GetSnapshotRequest) (*computepb.Snapshot, error) {
	if req.Project == "" || req.Snapshot == "" {
		return nil, invalidArg("project and snapshot are required")
	}
	s.state.mu.RLock()
	snap, ok := s.state.snapshots[resourceKey(req.Project, req.Snapshot)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("snapshot", req.Project, req.Snapshot)
	}
	return proto.Clone(snap).(*computepb.Snapshot), nil
}

func (s *snapshotsServer) Insert(_ context.Context, req *computepb.InsertSnapshotRequest) (*computepb.Operation, error) {
	snap := req.SnapshotResource
	if snap == nil || derefString(snap.Name) == "" {
		return nil, invalidArg("snapshot resource with a name is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := derefString(snap.Name)
	key := resourceKey(req.Project, name)

	s.state.mu.Lock()
	if _, exists := s.state.snapshots[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("snapshot", req.Project, name)
	}
	snap = proto.Clone(snap).(*computepb.Snapshot)
	snap.Id = protoUint64(s.state.nextID())
	snap.Kind = protoString("compute#snapshot")
	snap.Name = protoString(name)
	snap.CreationTimestamp = protoString(rfc3339Now())
	snap.SelfLink = protoString(globalSelfLink(req.Project, "global/snapshots", name))
	if snap.Status == nil {
		snap.Status = protoString("READY")
	}
	s.state.snapshots[key] = snap
	op := s.state.addOperation("projects/"+req.Project, "insert", snap.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *snapshotsServer) Delete(_ context.Context, req *computepb.DeleteSnapshotRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Snapshot == "" {
		return nil, invalidArg("project and snapshot are required")
	}
	key := resourceKey(req.Project, req.Snapshot)
	s.state.mu.Lock()
	snap, ok := s.state.snapshots[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("snapshot", req.Project, req.Snapshot)
	}
	target := snap.GetSelfLink()
	delete(s.state.snapshots, key)
	op := s.state.addOperation("projects/"+req.Project, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *snapshotsServer) List(_ context.Context, req *computepb.ListSnapshotsRequest) (*computepb.SnapshotList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	var items []*computepb.Snapshot
	prefix := req.Project + "|"
	for key, snap := range s.state.snapshots {
		if strings.HasPrefix(key, prefix) {
			items = append(items, snap)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.SnapshotList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/global/snapshots", selfLinkBase, req.Project)),
		Kind:          protoString("compute#snapshotList"),
	}, nil
}

// ---------------------------------------------------------------------------
// Instance templates (global)
// ---------------------------------------------------------------------------

func (s *instanceTemplatesServer) Get(_ context.Context, req *computepb.GetInstanceTemplateRequest) (*computepb.InstanceTemplate, error) {
	if req.Project == "" || req.InstanceTemplate == "" {
		return nil, invalidArg("project and instance template are required")
	}
	s.state.mu.RLock()
	t, ok := s.state.instanceTpls[resourceKey(req.Project, req.InstanceTemplate)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("instance template", req.Project, req.InstanceTemplate)
	}
	return proto.Clone(t).(*computepb.InstanceTemplate), nil
}

func (s *instanceTemplatesServer) Insert(_ context.Context, req *computepb.InsertInstanceTemplateRequest) (*computepb.Operation, error) {
	t := req.InstanceTemplateResource
	if t == nil || derefString(t.Name) == "" {
		return nil, invalidArg("instance template resource with a name is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := derefString(t.Name)
	key := resourceKey(req.Project, name)

	s.state.mu.Lock()
	if _, exists := s.state.instanceTpls[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("instance template", req.Project, name)
	}
	t = proto.Clone(t).(*computepb.InstanceTemplate)
	t.Id = protoUint64(s.state.nextID())
	t.Kind = protoString("compute#instanceTemplate")
	t.Name = protoString(name)
	t.CreationTimestamp = protoString(rfc3339Now())
	t.SelfLink = protoString(globalSelfLink(req.Project, "instanceTemplates", name))
	s.state.instanceTpls[key] = t
	op := s.state.addOperation("projects/"+req.Project, "insert", t.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *instanceTemplatesServer) Delete(_ context.Context, req *computepb.DeleteInstanceTemplateRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.InstanceTemplate == "" {
		return nil, invalidArg("project and instance template are required")
	}
	key := resourceKey(req.Project, req.InstanceTemplate)
	s.state.mu.Lock()
	t, ok := s.state.instanceTpls[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("instance template", req.Project, req.InstanceTemplate)
	}
	target := t.GetSelfLink()
	delete(s.state.instanceTpls, key)
	op := s.state.addOperation("projects/"+req.Project, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *instanceTemplatesServer) List(_ context.Context, req *computepb.ListInstanceTemplatesRequest) (*computepb.InstanceTemplateList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	var items []*computepb.InstanceTemplate
	prefix := req.Project + "|"
	for key, t := range s.state.instanceTpls {
		if strings.HasPrefix(key, prefix) {
			items = append(items, t)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.InstanceTemplateList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/instanceTemplates", selfLinkBase, req.Project)),
		Kind:          protoString("compute#instanceTemplateList"),
	}, nil
}

func (s *instanceTemplatesServer) AggregatedList(_ context.Context, req *computepb.AggregatedListInstanceTemplatesRequest) (*computepb.InstanceTemplateAggregatedList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	groups := map[string][]*computepb.InstanceTemplate{}
	for key, t := range s.state.instanceTpls {
		parts := strings.Split(key, "|")
		if len(parts) < 2 || parts[0] != req.Project {
			continue
		}
		scope := "global"
		if t.Region != nil && *t.Region != "" {
			scope = "regions/" + regionName(*t.Region)
		}
		groups[scope] = append(groups[scope], t)
	}
	s.state.mu.RUnlock()

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := &computepb.InstanceTemplateAggregatedList{
		Id:    protoString("compute.instanceTemplates.aggregatedList"),
		Kind:  protoString("compute#instanceTemplateAggregatedList"),
		Items: map[string]*computepb.InstanceTemplatesScopedList{},
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/aggregated/instanceTemplates",
			selfLinkBase, req.Project)),
	}
	for _, scope := range keys {
		items := groups[scope]
		byName(items)
		out.Items[scope] = &computepb.InstanceTemplatesScopedList{
			InstanceTemplates: items,
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Health checks (global or regional)
// ---------------------------------------------------------------------------

func healthCheckKey(project, region, name string) string {
	return resourceKey(project, regionName(region), name)
}

func (s *healthChecksServer) Get(_ context.Context, req *computepb.GetHealthCheckRequest) (*computepb.HealthCheck, error) {
	if req.Project == "" || req.HealthCheck == "" {
		return nil, invalidArg("project and health check are required")
	}
	s.state.mu.RLock()
	// Health checks are looked up across regions: try each stored scope.
	for key, hc := range s.state.healthChecks {
		parts := strings.Split(key, "|")
		if len(parts) >= 2 && parts[0] == req.Project && parts[len(parts)-1] == req.HealthCheck {
			s.state.mu.RUnlock()
			return proto.Clone(hc).(*computepb.HealthCheck), nil
		}
	}
	s.state.mu.RUnlock()
	return nil, notFound("health check", req.Project, req.HealthCheck)
}

func (s *healthChecksServer) Insert(_ context.Context, req *computepb.InsertHealthCheckRequest) (*computepb.Operation, error) {
	hc := req.HealthCheckResource
	if hc == nil || derefString(hc.Name) == "" {
		return nil, invalidArg("health check resource with a name is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := derefString(hc.Name)
	region := regionName(derefString(hc.Region))
	key := healthCheckKey(req.Project, region, name)

	s.state.mu.Lock()
	if _, exists := s.state.healthChecks[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("health check", req.Project, name)
	}
	hc = proto.Clone(hc).(*computepb.HealthCheck)
	hc.Id = protoUint64(s.state.nextID())
	hc.Kind = protoString("compute#healthCheck")
	hc.Name = protoString(name)
	hc.CreationTimestamp = protoString(rfc3339Now())
	if region != "" {
		hc.SelfLink = protoString(fmt.Sprintf("%s/projects/%s/regions/%s/healthChecks/%s",
			selfLinkBase, req.Project, region, name))
	} else {
		hc.SelfLink = protoString(globalSelfLink(req.Project, "healthChecks", name))
	}
	s.state.healthChecks[key] = hc
	op := s.state.addOperation("projects/"+req.Project, "insert", hc.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *healthChecksServer) Delete(_ context.Context, req *computepb.DeleteHealthCheckRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.HealthCheck == "" {
		return nil, invalidArg("project and health check are required")
	}
	s.state.mu.Lock()
	var foundKey string
	var found *computepb.HealthCheck
	for key, hc := range s.state.healthChecks {
		parts := strings.Split(key, "|")
		if len(parts) >= 2 && parts[0] == req.Project && parts[len(parts)-1] == req.HealthCheck {
			foundKey, found = key, hc
			break
		}
	}
	if found == nil {
		s.state.mu.Unlock()
		return nil, notFound("health check", req.Project, req.HealthCheck)
	}
	target := found.GetSelfLink()
	delete(s.state.healthChecks, foundKey)
	op := s.state.addOperation("projects/"+req.Project, "delete", target)
	s.state.mu.Unlock()
	return op, nil
}

func (s *healthChecksServer) List(_ context.Context, req *computepb.ListHealthChecksRequest) (*computepb.HealthCheckList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	var items []*computepb.HealthCheck
	for key, hc := range s.state.healthChecks {
		parts := strings.Split(key, "|")
		if len(parts) >= 2 && parts[0] == req.Project {
			items = append(items, hc)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.HealthCheckList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/healthChecks", selfLinkBase, req.Project)),
		Kind:          protoString("compute#healthCheckList"),
	}, nil
}

func (s *healthChecksServer) Patch(_ context.Context, req *computepb.PatchHealthCheckRequest) (*computepb.Operation, error) {
	if req.HealthCheckResource == nil {
		return nil, invalidArg("health check resource is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := req.GetHealthCheck()
	if name == "" {
		name = derefString(req.HealthCheckResource.Name)
	}
	if name == "" {
		return nil, invalidArg("health check name is required")
	}
	s.state.mu.Lock()
	var foundKey string
	var stored *computepb.HealthCheck
	for key, hc := range s.state.healthChecks {
		parts := strings.Split(key, "|")
		if len(parts) >= 2 && parts[0] == req.Project && parts[len(parts)-1] == name {
			foundKey, stored = key, hc
			break
		}
	}
	if stored == nil {
		s.state.mu.Unlock()
		return nil, notFound("health check", req.Project, name)
	}
	patched := proto.Clone(req.HealthCheckResource).(*computepb.HealthCheck)
	patched.Id = stored.Id
	patched.CreationTimestamp = stored.CreationTimestamp
	patched.SelfLink = stored.SelfLink
	patched.Kind = stored.Kind
	patched.Name = stored.Name
	s.state.healthChecks[foundKey] = patched
	op := s.state.addOperation("projects/"+req.Project, "update", patched.GetSelfLink())
	s.state.mu.Unlock()
	return op, nil
}

func (s *healthChecksServer) AggregatedList(_ context.Context, req *computepb.AggregatedListHealthChecksRequest) (*computepb.HealthChecksAggregatedList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	s.state.mu.RLock()
	groups := map[string][]*computepb.HealthCheck{}
	for key, hc := range s.state.healthChecks {
		parts := strings.Split(key, "|")
		if len(parts) < 2 || parts[0] != req.Project {
			continue
		}
		scope := "global"
		if len(parts) >= 3 && parts[1] != "" {
			scope = "regions/" + parts[1]
		}
		groups[scope] = append(groups[scope], hc)
	}
	s.state.mu.RUnlock()

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := &computepb.HealthChecksAggregatedList{
		Id:    protoString("compute.healthChecks.aggregatedList"),
		Kind:  protoString("compute#healthCheckAggregatedList"),
		Items: map[string]*computepb.HealthChecksScopedList{},
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/aggregated/healthChecks",
			selfLinkBase, req.Project)),
	}
	for _, scope := range keys {
		items := groups[scope]
		byName(items)
		out.Items[scope] = &computepb.HealthChecksScopedList{
			HealthChecks: items,
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Machine types, zones, regions: reference data the emulator does not
// fabricate, so lists are empty and gets are NOT_FOUND.
// ---------------------------------------------------------------------------

func (s *machineTypesServer) Get(_ context.Context, req *computepb.GetMachineTypeRequest) (*computepb.MachineType, error) {
	return nil, notFound("machine type", req.Project, req.MachineType)
}

func (s *machineTypesServer) List(_ context.Context, req *computepb.ListMachineTypesRequest) (*computepb.MachineTypeList, error) {
	if req.Project == "" || req.Zone == "" {
		return nil, invalidArg("project and zone are required")
	}
	zone := zoneName(req.Zone)
	return &computepb.MachineTypeList{
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/zones/%s/machineTypes", selfLinkBase, req.Project, zone)),
		Kind:     protoString("compute#machineTypeList"),
	}, nil
}

func (s *machineTypesServer) AggregatedList(_ context.Context, req *computepb.AggregatedListMachineTypesRequest) (*computepb.MachineTypeAggregatedList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	return &computepb.MachineTypeAggregatedList{
		Id:       protoString("compute.machineTypes.aggregatedList"),
		Kind:     protoString("compute#machineTypeAggregatedList"),
		Items:    map[string]*computepb.MachineTypesScopedList{},
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/aggregated/machineTypes", selfLinkBase, req.Project)),
	}, nil
}

func (s *zonesServer) Get(_ context.Context, req *computepb.GetZoneRequest) (*computepb.Zone, error) {
	return nil, notFound("zone", req.Project, req.Zone)
}

func (s *zonesServer) List(_ context.Context, req *computepb.ListZonesRequest) (*computepb.ZoneList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	return &computepb.ZoneList{
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/zones", selfLinkBase, req.Project)),
		Kind:     protoString("compute#zoneList"),
	}, nil
}

func (s *regionsServer) Get(_ context.Context, req *computepb.GetRegionRequest) (*computepb.Region, error) {
	return nil, notFound("region", req.Project, req.Region)
}

func (s *regionsServer) List(_ context.Context, req *computepb.ListRegionsRequest) (*computepb.RegionList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	return &computepb.RegionList{
		SelfLink: protoString(fmt.Sprintf("%s/projects/%s/regions", selfLinkBase, req.Project)),
		Kind:     protoString("compute#regionList"),
	}, nil
}

// ---------------------------------------------------------------------------
// Operations
// ---------------------------------------------------------------------------

func (s *globalOperationsServer) Get(_ context.Context, req *computepb.GetGlobalOperationRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Operation == "" {
		return nil, invalidArg("project and operation are required")
	}
	op, ok := s.state.getOperation("projects/"+req.Project, req.Operation)
	if !ok {
		return nil, notFound("operation", req.Project, req.Operation)
	}
	return proto.Clone(op).(*computepb.Operation), nil
}

func (s *globalOperationsServer) List(_ context.Context, req *computepb.ListGlobalOperationsRequest) (*computepb.OperationList, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	scope := "projects/" + req.Project
	items := s.state.listOperations(scope)
	// Global operations have no zone/region segment.
	filtered := items[:0]
	for _, op := range items {
		name := op.GetName()
		rest := strings.TrimPrefix(name, scope+"/operations/")
		if !strings.Contains(rest, "/") {
			filtered = append(filtered, op)
		}
	}
	page, next := paginate(filtered, req.MaxResults, req.PageToken)
	return &computepb.OperationList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/globalOperations", selfLinkBase, req.Project)),
		Kind:          protoString("compute#operationList"),
	}, nil
}

func (s *zoneOperationsServer) Get(_ context.Context, req *computepb.GetZoneOperationRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Zone == "" || req.Operation == "" {
		return nil, invalidArg("project, zone and operation are required")
	}
	zone := zoneName(req.Zone)
	op, ok := s.state.getOperation("projects/"+req.Project+"/zones/"+zone, req.Operation)
	if !ok {
		return nil, notFound("operation", req.Project, req.Operation)
	}
	return proto.Clone(op).(*computepb.Operation), nil
}

func (s *zoneOperationsServer) List(_ context.Context, req *computepb.ListZoneOperationsRequest) (*computepb.OperationList, error) {
	if req.Project == "" || req.Zone == "" {
		return nil, invalidArg("project and zone are required")
	}
	zone := zoneName(req.Zone)
	items := s.state.listOperations("projects/" + req.Project + "/zones/" + zone)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.OperationList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/zones/%s/operations", selfLinkBase, req.Project, zone)),
		Kind:          protoString("compute#operationList"),
	}, nil
}

func (s *regionOperationsServer) Get(_ context.Context, req *computepb.GetRegionOperationRequest) (*computepb.Operation, error) {
	if req.Project == "" || req.Region == "" || req.Operation == "" {
		return nil, invalidArg("project, region and operation are required")
	}
	region := regionName(req.Region)
	op, ok := s.state.getOperation("projects/"+req.Project+"/regions/"+region, req.Operation)
	if !ok {
		return nil, notFound("operation", req.Project, req.Operation)
	}
	return proto.Clone(op).(*computepb.Operation), nil
}

func (s *regionOperationsServer) List(_ context.Context, req *computepb.ListRegionOperationsRequest) (*computepb.OperationList, error) {
	if req.Project == "" || req.Region == "" {
		return nil, invalidArg("project and region are required")
	}
	region := regionName(req.Region)
	items := s.state.listOperations("projects/" + req.Project + "/regions/" + region)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &computepb.OperationList{
		Items:         page,
		NextPageToken: nextTokenPtr(next),
		SelfLink:      protoString(fmt.Sprintf("%s/projects/%s/regions/%s/operations", selfLinkBase, req.Project, region)),
		Kind:          protoString("compute#operationList"),
	}, nil
}
