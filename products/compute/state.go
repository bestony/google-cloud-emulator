// Package compute emulates the Google Compute Engine v1 control plane:
// resource CRUD for instances, disks, networks, subnetworks, firewalls,
// addresses, images, snapshots, instance templates and health checks, plus
// zonal/regional/global operation tracking. Pure in-memory.
package compute

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	computepb "github.com/bestony/google-cloud-emulator/gen/google/cloud/compute/v1"
)

const selfLinkBase = "https://www.googleapis.com/compute/v1"

type State struct {
	mu sync.RWMutex

	instances    map[string]*computepb.Instance
	disks        map[string]*computepb.Disk
	networks     map[string]*computepb.Network
	subnetworks  map[string]*computepb.Subnetwork
	firewalls    map[string]*computepb.Firewall
	addresses    map[string]*computepb.Address
	images       map[string]*computepb.Image
	snapshots    map[string]*computepb.Snapshot
	instanceTpls map[string]*computepb.InstanceTemplate
	healthChecks map[string]*computepb.HealthCheck
	operations   map[string]*computepb.Operation

	nextOperationID uint64
	nextResourceID  uint64
}

func NewState() *State {
	return &State{
		instances:    map[string]*computepb.Instance{},
		disks:        map[string]*computepb.Disk{},
		networks:     map[string]*computepb.Network{},
		subnetworks:  map[string]*computepb.Subnetwork{},
		firewalls:    map[string]*computepb.Firewall{},
		addresses:    map[string]*computepb.Address{},
		images:       map[string]*computepb.Image{},
		snapshots:    map[string]*computepb.Snapshot{},
		instanceTpls: map[string]*computepb.InstanceTemplate{},
		healthChecks: map[string]*computepb.HealthCheck{},
		operations:   map[string]*computepb.Operation{},
	}
}

// resourceKey builds a collision-free key from scope parts.
func resourceKey(parts ...string) string { return strings.Join(parts, "|") }

func (s *State) nextID() uint64 {
	s.nextResourceID++
	return s.nextResourceID
}

func rfc3339Now() string { return time.Now().UTC().Format(time.RFC3339) }

// zoneName normalizes "zones/us-central1-a", "us-central1-a" or a full URL to
// the bare zone name.
func zoneName(zone string) string {
	zone = strings.TrimSuffix(zone, "/")
	if i := strings.LastIndex(zone, "/"); i >= 0 {
		zone = zone[i+1:]
	}
	return zone
}

func regionName(region string) string {
	region = strings.TrimSuffix(region, "/")
	if i := strings.LastIndex(region, "/"); i >= 0 {
		region = region[i+1:]
	}
	return region
}

func zoneSelfLink(project, zone string) string {
	return fmt.Sprintf("%s/projects/%s/zones/%s", selfLinkBase, project, zone)
}

func regionSelfLink(project, region string) string {
	return fmt.Sprintf("%s/projects/%s/regions/%s", selfLinkBase, project, region)
}

func globalSelfLink(project, kind, name string) string {
	return fmt.Sprintf("%s/projects/%s/%s/%s", selfLinkBase, project, kind, name)
}

// addOperation records a completed operation. scopePath is
// "projects/{p}", "projects/{p}/zones/{z}" or "projects/{p}/regions/{r}".
// Callers must hold s.mu (write lock).
func (s *State) addOperation(scopePath, opType, targetLink string) *computepb.Operation {
	s.nextOperationID++
	id := s.nextOperationID
	name := fmt.Sprintf("%s/operations/%d", scopePath, id)
	ts := rfc3339Now()
	op := &computepb.Operation{
		Id:            protoUint64(id),
		Name:          protoString(name),
		OperationType: protoString(opType),
		TargetLink:    protoString(targetLink),
		Status:        computepb.Operation_DONE.Enum(),
		Progress:      protoInt32(100),
		StartTime:     protoString(ts),
		EndTime:       protoString(ts),
		Kind:          protoString("compute#operation"),
		SelfLink:      protoString(selfLinkBase + "/" + name),
	}
	s.operations[name] = op
	return op
}

// getOperation finds an operation by id within a scope. reqID may be a bare
// id or a full resource name.
func (s *State) getOperation(scopePath, reqID string) (*computepb.Operation, bool) {
	id := reqID
	if i := strings.LastIndex(reqID, "/"); i >= 0 {
		id = reqID[i+1:]
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for name, op := range s.operations {
		if strings.HasPrefix(name, scopePath+"/operations/") && name == fmt.Sprintf("%s/operations/%s", scopePath, id) {
			return op, true
		}
	}
	return nil, false
}

// listOperations returns operations whose names start with scopePath.
func (s *State) listOperations(scopePath string) []*computepb.Operation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := scopePath + "/operations/"
	var out []*computepb.Operation
	for name, op := range s.operations {
		if strings.HasPrefix(name, prefix) {
			out = append(out, op)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].GetId() < out[j].GetId()
	})
	return out
}
