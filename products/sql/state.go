// Package sql emulates the Google Cloud SQL Admin API v1 control plane:
// database instance CRUD, users, databases, backup runs and operation
// tracking. Pure in-memory; unimplemented surfaces stay UNIMPLEMENTED via
// the embedded generated servers.
package sql

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	sqlpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/sql/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const selfLinkBase = "https://sqladmin.googleapis.com/sql/v1"

type State struct {
	mu sync.RWMutex

	// Instances are keyed project|name (names are unique per project).
	instances  map[string]*sqlpb.DatabaseInstance
	users      map[string]*sqlpb.User
	databases  map[string]*sqlpb.Database
	backupRuns map[string]*sqlpb.BackupRun
	operations map[string]*sqlpb.Operation

	nextID uint64
}

func NewState() *State {
	return &State{
		instances:  map[string]*sqlpb.DatabaseInstance{},
		users:      map[string]*sqlpb.User{},
		databases:  map[string]*sqlpb.Database{},
		backupRuns: map[string]*sqlpb.BackupRun{},
		operations: map[string]*sqlpb.Operation{},
	}
}

// nextIDLocked returns a unique id. Caller must hold s.mu.
func (s *State) nextIDLocked() uint64 {
	s.nextID++
	return s.nextID
}

func instanceKey(project, name string) string { return project + "|" + name }

func userKey(project, instance, name, host string) string {
	return strings.Join([]string{project, instance, name, host}, "|")
}

func databaseKey(project, instance, name string) string {
	return strings.Join([]string{project, instance, name}, "|")
}

func backupRunKey(project, instance string, id int64) string {
	return fmt.Sprintf("%s|%s|%d", project, instance, id)
}

func instanceSelfLink(project, name string) string {
	return fmt.Sprintf("%s/projects/%s/instances/%s", selfLinkBase, project, name)
}

func userSelfLink(project, instance, name string) string {
	return fmt.Sprintf("%s/projects/%s/instances/%s/users/%s", selfLinkBase, project, instance, name)
}

func databaseSelfLink(project, instance, name string) string {
	return fmt.Sprintf("%s/projects/%s/instances/%s/databases/%s", selfLinkBase, project, instance, name)
}

func backupRunSelfLink(project, instance string, id int64) string {
	return fmt.Sprintf("%s/projects/%s/instances/%s/backupRuns/%d", selfLinkBase, project, instance, id)
}

func operationSelfLink(project, name string) string {
	return fmt.Sprintf("%s/projects/%s/operations/%s", selfLinkBase, project, name)
}

// addOperation records a completed operation. Caller must hold s.mu.
func (s *State) addOperation(project string, opType sqlpb.Operation_SqlOperationType, targetID, targetLink string) *sqlpb.Operation {
	id := s.nextIDLocked()
	name := fmt.Sprintf("%d", id)
	now := timestamppb.Now()
	op := &sqlpb.Operation{
		Kind:          "sql#operation",
		Name:          name,
		OperationType: opType,
		Status:        sqlpb.Operation_DONE,
		TargetId:      targetID,
		TargetProject: project,
		TargetLink:    targetLink,
		SelfLink:      operationSelfLink(project, name),
		InsertTime:    now,
		StartTime:     now,
		EndTime:       now,
	}
	s.operations[instanceKey(project, name)] = op
	return op
}

func (s *State) getOperation(project, name string) (*sqlpb.Operation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	op, ok := s.operations[instanceKey(project, name)]
	return op, ok
}

// listOperations returns operations for a project, optionally filtered by
// instance name and location. Caller must NOT hold s.mu.
func (s *State) listOperations(project, instance, location string) []*sqlpb.Operation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*sqlpb.Operation
	prefix := project + "|"
	for key, op := range s.operations {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if instance != "" && op.GetTargetId() != instance {
			continue
		}
		if location != "" {
			// Match the operation's target instance region/zone.
			inst, ok := s.instances[instanceKey(project, op.GetTargetId())]
			if !ok || (inst.GetRegion() != location && inst.GetGceZone() != location) {
				continue
			}
		}
		out = append(out, op)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetName() < out[j].GetName() })
	return out
}
