package sql

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	sqlpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/sql/v1"
)

func notFound(what, project, name string) error {
	return status.Errorf(codes.NotFound, "%s %q not found in project %q", what, name, project)
}

func invalidArg(msg string) error {
	return status.Errorf(codes.InvalidArgument, "%s", msg)
}

func alreadyExists(what, project, name string) error {
	return status.Errorf(codes.AlreadyExists, "%s %q already exists in project %q", what, name, project)
}

// paginate applies offset pagination: pageToken is a decimal offset.
func paginate[T any](items []T, maxResults uint32, pageToken string) ([]T, string) {
	offset := 0
	if pageToken != "" {
		offset, _ = strconv.Atoi(pageToken)
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(items) {
		offset = len(items)
	}
	items = items[offset:]
	if maxResults > 0 && uint32(len(items)) > maxResults {
		return items[:maxResults], strconv.Itoa(offset + int(maxResults))
	}
	return items, ""
}

func byName[T interface{ GetName() string }](items []T) {
	sort.Slice(items, func(i, j int) bool { return items[i].GetName() < items[j].GetName() })
}

func nextToken(next string) string { return next }

// ---------------------------------------------------------------------------
// Instances
// ---------------------------------------------------------------------------

type instancesServer struct {
	sqlpb.UnimplementedSqlInstancesServiceServer
	state *State
}

func (s *instancesServer) Get(_ context.Context, req *sqlpb.SqlInstancesGetRequest) (*sqlpb.DatabaseInstance, error) {
	if req.Project == "" || req.Instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	s.state.mu.RLock()
	inst, ok := s.state.instances[instanceKey(req.Project, req.Instance)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("instance", req.Project, req.Instance)
	}
	return proto.Clone(inst).(*sqlpb.DatabaseInstance), nil
}

func (s *instancesServer) Insert(_ context.Context, req *sqlpb.SqlInstancesInsertRequest) (*sqlpb.Operation, error) {
	body := req.Body
	if body == nil || body.Name == "" {
		return nil, invalidArg("instance body with a name is required")
	}
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	name := body.Name
	key := instanceKey(req.Project, name)

	s.state.mu.Lock()
	if _, exists := s.state.instances[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("instance", req.Project, name)
	}
	inst := proto.Clone(body).(*sqlpb.DatabaseInstance)
	inst.Kind = "sql#instance"
	inst.Name = name
	inst.State = sqlpb.DatabaseInstance_RUNNABLE
	inst.Etag = strconv.FormatUint(s.state.nextIDLocked(), 10)
	inst.SelfLink = instanceSelfLink(req.Project, name)
	inst.CreateTime = timestamppb.Now()
	if inst.Region == "" {
		inst.Region = req.Location
	}
	s.state.instances[key] = inst
	op := s.state.addOperation(req.Project, sqlpb.Operation_CREATE, name, inst.SelfLink)
	s.state.mu.Unlock()
	return op, nil
}

func (s *instancesServer) List(_ context.Context, req *sqlpb.SqlInstancesListRequest) (*sqlpb.InstancesListResponse, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	if req.Filter != "" {
		return nil, invalidArg("filter is not supported by the emulator")
	}
	s.state.mu.RLock()
	var items []*sqlpb.DatabaseInstance
	prefix := req.Project + "|"
	for key, inst := range s.state.instances {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if req.Location != "" && inst.GetRegion() != req.Location && inst.GetGceZone() != req.Location {
			continue
		}
		items = append(items, inst)
	}
	s.state.mu.RUnlock()
	byName(items)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &sqlpb.InstancesListResponse{
		Kind:          "sql#instancesList",
		Items:         page,
		NextPageToken: nextToken(next),
	}, nil
}

func (s *instancesServer) Patch(ctx context.Context, req *sqlpb.SqlInstancesPatchRequest) (*sqlpb.Operation, error) {
	return s.update(ctx, req.Project, req.Instance, req.Body, req.Location)
}

func (s *instancesServer) Update(ctx context.Context, req *sqlpb.SqlInstancesUpdateRequest) (*sqlpb.Operation, error) {
	return s.update(ctx, req.Project, req.Instance, req.Body, req.Location)
}

func (s *instancesServer) update(_ context.Context, project, name string, body *sqlpb.DatabaseInstance, _ string) (*sqlpb.Operation, error) {
	if project == "" || name == "" {
		return nil, invalidArg("project and instance are required")
	}
	if body == nil {
		return nil, invalidArg("instance body is required")
	}
	key := instanceKey(project, name)
	s.state.mu.Lock()
	stored, ok := s.state.instances[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("instance", project, name)
	}
	patched := proto.Clone(body).(*sqlpb.DatabaseInstance)
	patched.Kind = stored.Kind
	patched.Name = stored.Name
	patched.State = stored.State
	patched.Etag = strconv.FormatUint(s.state.nextIDLocked(), 10)
	patched.SelfLink = stored.SelfLink
	patched.CreateTime = stored.CreateTime
	if patched.Region == "" {
		patched.Region = stored.Region
	}
	s.state.instances[key] = patched
	op := s.state.addOperation(project, sqlpb.Operation_UPDATE, name, stored.SelfLink)
	s.state.mu.Unlock()
	return op, nil
}

func (s *instancesServer) Delete(_ context.Context, req *sqlpb.SqlInstancesDeleteRequest) (*sqlpb.Operation, error) {
	if req.Project == "" || req.Instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	key := instanceKey(req.Project, req.Instance)
	s.state.mu.Lock()
	inst, ok := s.state.instances[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("instance", req.Project, req.Instance)
	}
	target := inst.SelfLink
	delete(s.state.instances, key)
	op := s.state.addOperation(req.Project, sqlpb.Operation_DELETE, req.Instance, target)
	s.state.mu.Unlock()
	return op, nil
}

// ---------------------------------------------------------------------------
// Operations
// ---------------------------------------------------------------------------

type operationsServer struct {
	sqlpb.UnimplementedSqlOperationsServiceServer
	state *State
}

func (s *operationsServer) Get(_ context.Context, req *sqlpb.SqlOperationsGetRequest) (*sqlpb.Operation, error) {
	if req.Project == "" || req.Operation == "" {
		return nil, invalidArg("project and operation are required")
	}
	name := req.Operation
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	op, ok := s.state.getOperation(req.Project, name)
	if !ok {
		return nil, notFound("operation", req.Project, name)
	}
	return proto.Clone(op).(*sqlpb.Operation), nil
}

func (s *operationsServer) List(_ context.Context, req *sqlpb.SqlOperationsListRequest) (*sqlpb.OperationsListResponse, error) {
	if req.Project == "" {
		return nil, invalidArg("project is required")
	}
	items := s.state.listOperations(req.Project, req.Instance, req.Location)
	page, next := paginate(items, req.MaxResults, req.PageToken)
	return &sqlpb.OperationsListResponse{
		Kind:          "sql#operationsList",
		Items:         page,
		NextPageToken: nextToken(next),
	}, nil
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

type usersServer struct {
	sqlpb.UnimplementedSqlUsersServiceServer
	state *State
}

func requireInstance(st *State, project, instance string) error {
	st.mu.RLock()
	_, ok := st.instances[instanceKey(project, instance)]
	st.mu.RUnlock()
	if !ok {
		return notFound("instance", project, instance)
	}
	return nil
}

func (s *usersServer) findUser(project, instance, name, host string) (*sqlpb.User, string, bool) {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	var found *sqlpb.User
	var foundKey string
	prefix := strings.Join([]string{project, instance, name}, "|") + "|"
	for key, u := range s.state.users {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if host != "" && u.GetHost() != host {
			continue
		}
		if found == nil || u.GetHost() < found.GetHost() {
			found, foundKey = u, key
		}
	}
	return found, foundKey, found != nil
}

func (s *usersServer) Get(_ context.Context, req *sqlpb.SqlUsersGetRequest) (*sqlpb.User, error) {
	if req.Project == "" || req.Instance == "" || req.Name == "" {
		return nil, invalidArg("project, instance and name are required")
	}
	u, _, ok := s.findUser(req.Project, req.Instance, req.Name, req.Host)
	if !ok {
		return nil, notFound("user", req.Project, req.Name)
	}
	return proto.Clone(u).(*sqlpb.User), nil
}

func (s *usersServer) Insert(_ context.Context, req *sqlpb.SqlUsersInsertRequest) (*sqlpb.Operation, error) {
	body := req.Body
	if body == nil || body.Name == "" {
		return nil, invalidArg("user body with a name is required")
	}
	if req.Project == "" || req.Instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	if err := requireInstance(s.state, req.Project, req.Instance); err != nil {
		return nil, err
	}
	name := body.Name
	host := body.Host
	key := userKey(req.Project, req.Instance, name, host)

	s.state.mu.Lock()
	if _, exists := s.state.users[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("user", req.Project, name)
	}
	u := proto.Clone(body).(*sqlpb.User)
	u.Kind = "sql#user"
	u.Name = name
	u.Instance = req.Instance
	u.Project = req.Project
	u.Etag = strconv.FormatUint(s.state.nextIDLocked(), 10)
	s.state.users[key] = u
	op := s.state.addOperation(req.Project, sqlpb.Operation_CREATE, req.Instance, userSelfLink(req.Project, req.Instance, name))
	s.state.mu.Unlock()
	return op, nil
}

func (s *usersServer) List(_ context.Context, req *sqlpb.SqlUsersListRequest) (*sqlpb.UsersListResponse, error) {
	if req.Project == "" || req.Instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	prefix := strings.Join([]string{req.Project, req.Instance}, "|") + "|"
	s.state.mu.RLock()
	var items []*sqlpb.User
	for key, u := range s.state.users {
		if strings.HasPrefix(key, prefix) {
			items = append(items, u)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	return &sqlpb.UsersListResponse{
		Kind:  "sql#usersList",
		Items: items,
	}, nil
}

func (s *usersServer) Update(_ context.Context, req *sqlpb.SqlUsersUpdateRequest) (*sqlpb.Operation, error) {
	if req.Project == "" || req.Instance == "" || req.Name == "" {
		return nil, invalidArg("project, instance and name are required")
	}
	s.state.mu.Lock()
	u, key, ok := s.findUserLocked(req.Project, req.Instance, req.Name, req.Host)
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("user", req.Project, req.Name)
	}
	stored := proto.Clone(u).(*sqlpb.User)
	if len(req.DatabaseRoles) > 0 || (req.RevokeExistingRoles != nil && *req.RevokeExistingRoles) {
		stored.DatabaseRoles = append([]string(nil), req.DatabaseRoles...)
	}
	if len(req.ServerRoles) > 0 || (req.RevokeExistingServerRoles != nil && *req.RevokeExistingServerRoles) {
		stored.ServerRoles = append([]string(nil), req.ServerRoles...)
	}
	if req.Body != nil {
		if req.Body.Password != "" {
			stored.Password = req.Body.Password
		}
		if req.Body.Host != "" {
			// Host change moves the user to a new key.
			delete(s.state.users, key)
			stored.Host = req.Body.Host
			key = userKey(req.Project, req.Instance, req.Name, req.Body.Host)
		}
	}
	stored.Etag = strconv.FormatUint(s.state.nextIDLocked(), 10)
	s.state.users[key] = stored
	op := s.state.addOperation(req.Project, sqlpb.Operation_UPDATE, req.Instance, userSelfLink(req.Project, req.Instance, req.Name))
	s.state.mu.Unlock()
	return op, nil
}

// findUserLocked is findUser for callers holding s.mu (write lock).
func (s *usersServer) findUserLocked(project, instance, name, host string) (*sqlpb.User, string, bool) {
	var found *sqlpb.User
	var foundKey string
	prefix := strings.Join([]string{project, instance, name}, "|") + "|"
	for key, u := range s.state.users {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if host != "" && u.GetHost() != host {
			continue
		}
		if found == nil || u.GetHost() < found.GetHost() {
			found, foundKey = u, key
		}
	}
	return found, foundKey, found != nil
}

func (s *usersServer) Delete(_ context.Context, req *sqlpb.SqlUsersDeleteRequest) (*sqlpb.Operation, error) {
	if req.Project == "" || req.Instance == "" || req.Name == "" {
		return nil, invalidArg("project, instance and name are required")
	}
	s.state.mu.Lock()
	_, key, ok := s.findUserLocked(req.Project, req.Instance, req.Name, req.Host)
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("user", req.Project, req.Name)
	}
	target := userSelfLink(req.Project, req.Instance, req.Name)
	delete(s.state.users, key)
	op := s.state.addOperation(req.Project, sqlpb.Operation_DELETE, req.Instance, target)
	s.state.mu.Unlock()
	return op, nil
}

// ---------------------------------------------------------------------------
// Databases
// ---------------------------------------------------------------------------

type databasesServer struct {
	sqlpb.UnimplementedSqlDatabasesServiceServer
	state *State
}

func (s *databasesServer) Get(_ context.Context, req *sqlpb.SqlDatabasesGetRequest) (*sqlpb.Database, error) {
	if req.Project == "" || req.Instance == "" || req.Database == "" {
		return nil, invalidArg("project, instance and database are required")
	}
	s.state.mu.RLock()
	db, ok := s.state.databases[databaseKey(req.Project, req.Instance, req.Database)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("database", req.Project, req.Database)
	}
	return proto.Clone(db).(*sqlpb.Database), nil
}

func (s *databasesServer) Insert(_ context.Context, req *sqlpb.SqlDatabasesInsertRequest) (*sqlpb.Operation, error) {
	body := req.Body
	if body == nil || body.Name == "" {
		return nil, invalidArg("database body with a name is required")
	}
	if req.Project == "" || req.Instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	if err := requireInstance(s.state, req.Project, req.Instance); err != nil {
		return nil, err
	}
	name := body.Name
	key := databaseKey(req.Project, req.Instance, name)

	s.state.mu.Lock()
	if _, exists := s.state.databases[key]; exists {
		s.state.mu.Unlock()
		return nil, alreadyExists("database", req.Project, name)
	}
	db := proto.Clone(body).(*sqlpb.Database)
	db.Kind = "sql#database"
	db.Name = name
	db.Instance = req.Instance
	db.Project = req.Project
	db.Etag = strconv.FormatUint(s.state.nextIDLocked(), 10)
	db.SelfLink = databaseSelfLink(req.Project, req.Instance, name)
	s.state.databases[key] = db
	op := s.state.addOperation(req.Project, sqlpb.Operation_CREATE, req.Instance, db.SelfLink)
	s.state.mu.Unlock()
	return op, nil
}

func (s *databasesServer) List(_ context.Context, req *sqlpb.SqlDatabasesListRequest) (*sqlpb.DatabasesListResponse, error) {
	if req.Project == "" || req.Instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	prefix := databaseKey(req.Project, req.Instance, "") + ""
	s.state.mu.RLock()
	var items []*sqlpb.Database
	for key, db := range s.state.databases {
		if strings.HasPrefix(key, prefix) {
			items = append(items, db)
		}
	}
	s.state.mu.RUnlock()
	byName(items)
	return &sqlpb.DatabasesListResponse{
		Kind:  "sql#databasesList",
		Items: items,
	}, nil
}

func (s *databasesServer) Update(_ context.Context, req *sqlpb.SqlDatabasesUpdateRequest) (*sqlpb.Operation, error) {
	if req.Project == "" || req.Instance == "" || req.Database == "" {
		return nil, invalidArg("project, instance and database are required")
	}
	if req.Body == nil {
		return nil, invalidArg("database body is required")
	}
	key := databaseKey(req.Project, req.Instance, req.Database)
	s.state.mu.Lock()
	stored, ok := s.state.databases[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("database", req.Project, req.Database)
	}
	patched := proto.Clone(req.Body).(*sqlpb.Database)
	patched.Kind = stored.Kind
	patched.Name = stored.Name
	patched.Instance = stored.Instance
	patched.Project = stored.Project
	patched.SelfLink = stored.SelfLink
	patched.Etag = strconv.FormatUint(s.state.nextIDLocked(), 10)
	s.state.databases[key] = patched
	op := s.state.addOperation(req.Project, sqlpb.Operation_UPDATE, req.Instance, stored.SelfLink)
	s.state.mu.Unlock()
	return op, nil
}

func (s *databasesServer) Delete(_ context.Context, req *sqlpb.SqlDatabasesDeleteRequest) (*sqlpb.Operation, error) {
	if req.Project == "" || req.Instance == "" || req.Database == "" {
		return nil, invalidArg("project, instance and database are required")
	}
	key := databaseKey(req.Project, req.Instance, req.Database)
	s.state.mu.Lock()
	db, ok := s.state.databases[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("database", req.Project, req.Database)
	}
	target := db.SelfLink
	delete(s.state.databases, key)
	op := s.state.addOperation(req.Project, sqlpb.Operation_DELETE, req.Instance, target)
	s.state.mu.Unlock()
	return op, nil
}

// ---------------------------------------------------------------------------
// Backup runs
// ---------------------------------------------------------------------------

type backupRunsServer struct {
	sqlpb.UnimplementedSqlBackupRunsServiceServer
	state *State
}

func (s *backupRunsServer) Get(_ context.Context, req *sqlpb.SqlBackupRunsGetRequest) (*sqlpb.BackupRun, error) {
	if req.Project == "" || req.Instance == "" || req.Id == 0 {
		return nil, invalidArg("project, instance and id are required")
	}
	s.state.mu.RLock()
	br, ok := s.state.backupRuns[backupRunKey(req.Project, req.Instance, req.Id)]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("backup run", req.Project, strconv.FormatInt(req.Id, 10))
	}
	return proto.Clone(br).(*sqlpb.BackupRun), nil
}

func (s *backupRunsServer) Insert(_ context.Context, req *sqlpb.SqlBackupRunsInsertRequest) (*sqlpb.Operation, error) {
	if req.Project == "" || req.Instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	if err := requireInstance(s.state, req.Project, req.Instance); err != nil {
		return nil, err
	}
	s.state.mu.Lock()
	id := int64(s.state.nextIDLocked())
	now := timestamppb.Now()
	br := &sqlpb.BackupRun{
		Kind:            "sql#backupRun",
		Id:              id,
		Instance:        req.Instance,
		Status:          sqlpb.SqlBackupRunStatus_SUCCESSFUL,
		Type:            sqlpb.SqlBackupRunType_ON_DEMAND,
		EnqueuedTime:    now,
		StartTime:       now,
		EndTime:         now,
		WindowStartTime: now,
		SelfLink:        backupRunSelfLink(req.Project, req.Instance, id),
	}
	if req.Body != nil {
		br.Description = req.Body.Description
		if req.Body.Type != sqlpb.SqlBackupRunType_SQL_BACKUP_RUN_TYPE_UNSPECIFIED {
			br.Type = req.Body.Type
		}
	}
	key := backupRunKey(req.Project, req.Instance, id)
	s.state.backupRuns[key] = br
	op := s.state.addOperation(req.Project, sqlpb.Operation_BACKUP, req.Instance, br.SelfLink)
	s.state.mu.Unlock()
	return op, nil
}

func (s *backupRunsServer) List(_ context.Context, req *sqlpb.SqlBackupRunsListRequest) (*sqlpb.BackupRunsListResponse, error) {
	if req.Project == "" || req.Instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	prefix := fmt.Sprintf("%s|%s|", req.Project, req.Instance)
	s.state.mu.RLock()
	var items []*sqlpb.BackupRun
	for key, br := range s.state.backupRuns {
		if strings.HasPrefix(key, prefix) {
			items = append(items, br)
		}
	}
	s.state.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].GetId() > items[j].GetId() })
	page, next := paginate(items, uint32(req.MaxResults), req.PageToken)
	return &sqlpb.BackupRunsListResponse{
		Kind:          "sql#backupRunsList",
		Items:         page,
		NextPageToken: nextToken(next),
	}, nil
}

func (s *backupRunsServer) Delete(_ context.Context, req *sqlpb.SqlBackupRunsDeleteRequest) (*sqlpb.Operation, error) {
	if req.Project == "" || req.Instance == "" || req.Id == 0 {
		return nil, invalidArg("project, instance and id are required")
	}
	key := backupRunKey(req.Project, req.Instance, req.Id)
	s.state.mu.Lock()
	br, ok := s.state.backupRuns[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("backup run", req.Project, strconv.FormatInt(req.Id, 10))
	}
	target := br.SelfLink
	delete(s.state.backupRuns, key)
	op := s.state.addOperation(req.Project, sqlpb.Operation_DELETE, req.Instance, target)
	s.state.mu.Unlock()
	return op, nil
}
