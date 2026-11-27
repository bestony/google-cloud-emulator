// Package gke implements the google.container.v1 ClusterManager control
// plane: cluster CRUD, cluster update, delete, and operation get/list.
// Node pool, server config and certificate RPCs remain UNIMPLEMENTED via
// the embedded generated server. Pure in-memory state.
package gke

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	containerpb "github.com/bestony/google-cloud-emulator/gen/google/container/v1"
)

const selfLinkBase = "https://container.googleapis.com/v1"

// emulatorVersion is the static GKE version reported for created clusters.
const emulatorVersion = "1.31.4-gke.1250000"

type State struct {
	mu         sync.RWMutex
	clusters   map[string]*containerpb.Cluster
	operations map[string]*containerpb.Operation
	nextID     uint64
}

func NewState() *State {
	return &State{
		clusters:   make(map[string]*containerpb.Cluster),
		operations: make(map[string]*containerpb.Operation),
	}
}

func clusterKey(project, location, name string) string {
	return strings.Join([]string{project, location, name}, "|")
}

func operationKey(project, location, name string) string {
	return strings.Join([]string{project, location, name}, "|")
}

func clusterSelfLink(project, location, name string) string {
	return fmt.Sprintf("%s/projects/%s/locations/%s/clusters/%s", selfLinkBase, project, location, name)
}

func operationSelfLink(project, location, name string) string {
	return fmt.Sprintf("%s/projects/%s/locations/%s/operations/%s", selfLinkBase, project, location, name)
}

// nextIDLocked returns a fresh numeric id. Caller must hold s.mu (write).
func (s *State) nextIDLocked() uint64 {
	s.nextID++
	return s.nextID
}

// addOperation records a completed operation. Caller must hold s.mu (write).
func (s *State) addOperation(project, location string, opType containerpb.Operation_Type, targetLink, detail string) *containerpb.Operation {
	id := s.nextIDLocked()
	name := fmt.Sprintf("operation-%d", id)
	now := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	op := &containerpb.Operation{
		Name:          name,
		Zone:          location,
		Location:      location,
		OperationType: opType,
		Status:        containerpb.Operation_DONE,
		Detail:        detail,
		SelfLink:      operationSelfLink(project, location, name),
		TargetLink:    targetLink,
		StartTime:     now,
		EndTime:       now,
	}
	s.operations[operationKey(project, location, name)] = op
	return op
}

func (s *State) getOperation(project, location, name string) (*containerpb.Operation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	op, ok := s.operations[operationKey(project, location, name)]
	if !ok {
		return nil, false
	}
	return op, true
}

func (s *State) listOperations(project, location string) []*containerpb.Operation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*containerpb.Operation
	for key, op := range s.operations {
		parts := strings.Split(key, "|")
		if parts[0] != project {
			continue
		}
		if location != "" && parts[1] != location {
			continue
		}
		out = append(out, op)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetName() < out[j].GetName() })
	return out
}
