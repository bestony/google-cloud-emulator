package sql

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

	sqlpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/sql/v1"
)

const project = "demo"

func newInstances(s *State) *instancesServer   { return &instancesServer{state: s} }
func newOps(s *State) *operationsServer        { return &operationsServer{state: s} }
func newUsers(s *State) *usersServer           { return &usersServer{state: s} }
func newDBs(s *State) *databasesServer         { return &databasesServer{state: s} }
func newBackupRuns(s *State) *backupRunsServer { return &backupRunsServer{state: s} }

func insertInstance(t *testing.T, s *State, name, region string) *sqlpb.Operation {
	t.Helper()
	op, err := newInstances(s).Insert(context.Background(), &sqlpb.SqlInstancesInsertRequest{
		Project: project,
		Body: &sqlpb.DatabaseInstance{
			Name:            name,
			Region:          region,
			DatabaseVersion: sqlpb.SqlDatabaseVersion_MYSQL_8_0,
		},
	})
	if err != nil {
		t.Fatalf("Insert instance %s: %v", name, err)
	}
	return op
}

func TestInstanceLifecycle(t *testing.T) {
	st := NewState()
	srv := newInstances(st)
	ctx := context.Background()

	op := insertInstance(t, st, "db-1", "us-central1")
	if op.GetStatus() != sqlpb.Operation_DONE || op.GetOperationType() != sqlpb.Operation_CREATE {
		t.Fatalf("unexpected operation: %+v", op)
	}
	if op.GetName() == "" || op.GetTargetId() != "db-1" || op.GetSelfLink() == "" {
		t.Fatalf("operation missing fields: %+v", op)
	}

	// Duplicate -> AlreadyExists.
	if _, err := srv.Insert(ctx, &sqlpb.SqlInstancesInsertRequest{
		Project: project,
		Body:    &sqlpb.DatabaseInstance{Name: "db-1"},
	}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate insert: got %v", err)
	}

	inst, err := srv.Get(ctx, &sqlpb.SqlInstancesGetRequest{Project: project, Instance: "db-1"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if inst.GetName() != "db-1" || inst.GetKind() != "sql#instance" ||
		inst.GetState() != sqlpb.DatabaseInstance_RUNNABLE || inst.GetEtag() == "" ||
		inst.GetSelfLink() == "" || inst.GetCreateTime() == nil {
		t.Fatalf("unexpected instance: %+v", inst)
	}
	if inst.GetRegion() != "us-central1" {
		t.Fatalf("region = %q", inst.GetRegion())
	}

	// List.
	l, err := srv.List(ctx, &sqlpb.SqlInstancesListRequest{Project: project})
	if err != nil || len(l.Items) != 1 || l.Kind != "sql#instancesList" {
		t.Fatalf("List: %v %+v", err, l)
	}
	// Location filter.
	l2, err := srv.List(ctx, &sqlpb.SqlInstancesListRequest{Project: project, Location: "europe-west1"})
	if err != nil || len(l2.Items) != 0 {
		t.Fatalf("List location filter: %v %+v", err, l2)
	}
	// Unsupported filter is an honest error.
	if _, err := srv.List(ctx, &sqlpb.SqlInstancesListRequest{Project: project, Filter: "name:x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("filter: got %v", err)
	}

	// Patch preserves identity, bumps etag.
	oldEtag := inst.GetEtag()
	if _, err := srv.Patch(ctx, &sqlpb.SqlInstancesPatchRequest{
		Project:  project,
		Instance: "db-1",
		Body:     &sqlpb.DatabaseInstance{Settings: &sqlpb.Settings{Tier: "db-n1-standard-1"}},
	}); err != nil {
		t.Fatalf("Patch: %v", err)
	}
	patched, _ := srv.Get(ctx, &sqlpb.SqlInstancesGetRequest{Project: project, Instance: "db-1"})
	if patched.GetSettings().GetTier() != "db-n1-standard-1" || patched.GetEtag() == oldEtag ||
		patched.GetState() != sqlpb.DatabaseInstance_RUNNABLE || patched.GetSelfLink() != inst.GetSelfLink() {
		t.Fatalf("patch did not preserve identity: %+v", patched)
	}

	// Delete -> operation, then Get is NotFound.
	if _, err := srv.Delete(ctx, &sqlpb.SqlInstancesDeleteRequest{Project: project, Instance: "db-1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := srv.Get(ctx, &sqlpb.SqlInstancesGetRequest{Project: project, Instance: "db-1"}); status.Code(err) != codes.NotFound {
		t.Fatalf("get deleted: got %v", err)
	}
}

func TestOperations(t *testing.T) {
	st := NewState()
	ctx := context.Background()
	op := insertInstance(t, st, "db-1", "us-central1")

	got, err := newOps(st).Get(ctx, &sqlpb.SqlOperationsGetRequest{Project: project, Operation: op.GetName()})
	if err != nil || got.GetName() != op.GetName() {
		t.Fatalf("Get operation: %v %+v", err, got)
	}
	// Full self link also resolves.
	got2, err := newOps(st).Get(ctx, &sqlpb.SqlOperationsGetRequest{Project: project, Operation: op.GetSelfLink()})
	if err != nil || got2.GetName() != op.GetName() {
		t.Fatalf("Get operation by selfLink: %v %+v", err, got2)
	}
	if _, err := newOps(st).Get(ctx, &sqlpb.SqlOperationsGetRequest{Project: project, Operation: "999"}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing operation: got %v", err)
	}

	l, err := newOps(st).List(ctx, &sqlpb.SqlOperationsListRequest{Project: project, Instance: "db-1"})
	if err != nil || len(l.Items) != 1 || l.Kind != "sql#operationsList" {
		t.Fatalf("List operations: %v %+v", err, l)
	}
	lNone, err := newOps(st).List(ctx, &sqlpb.SqlOperationsListRequest{Project: project, Instance: "other"})
	if err != nil || len(lNone.Items) != 0 {
		t.Fatalf("List operations filter: %v %+v", err, lNone)
	}
	lLoc, err := newOps(st).List(ctx, &sqlpb.SqlOperationsListRequest{Project: project, Location: "us-central1"})
	if err != nil || len(lLoc.Items) != 1 {
		t.Fatalf("List operations location: %v %+v", err, lLoc)
	}

	// Cancel is honestly unimplemented (operations complete instantly).
	if _, err := newOps(st).Cancel(ctx, &sqlpb.SqlOperationsCancelRequest{Project: project, Operation: op.GetName()}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("Cancel: got %v, want Unimplemented", err)
	}
}

func TestUserDatabaseBackupRun(t *testing.T) {
	st := NewState()
	ctx := context.Background()
	insertInstance(t, st, "db-1", "us-central1")

	// User lifecycle.
	users := newUsers(st)
	op, err := users.Insert(ctx, &sqlpb.SqlUsersInsertRequest{
		Project:  project,
		Instance: "db-1",
		Body:     &sqlpb.User{Name: "alice", Password: "pw", Host: "%", Type: sqlpb.User_BUILT_IN},
	})
	if err != nil || op.GetOperationType() != sqlpb.Operation_CREATE {
		t.Fatalf("Insert user: %v %+v", err, op)
	}
	u, err := users.Get(ctx, &sqlpb.SqlUsersGetRequest{Project: project, Instance: "db-1", Name: "alice"})
	if err != nil || u.GetName() != "alice" || u.GetHost() != "%" || u.GetKind() != "sql#user" || u.GetEtag() == "" {
		t.Fatalf("Get user: %v %+v", err, u)
	}
	ul, err := users.List(ctx, &sqlpb.SqlUsersListRequest{Project: project, Instance: "db-1"})
	if err != nil || len(ul.Items) != 1 || ul.Kind != "sql#usersList" {
		t.Fatalf("List users: %v %+v", err, ul)
	}
	if _, err := users.Update(ctx, &sqlpb.SqlUsersUpdateRequest{
		Project: project, Instance: "db-1", Name: "alice",
		DatabaseRoles: []string{"mysql", "appdb"},
	}); err != nil {
		t.Fatalf("Update user: %v", err)
	}
	u2, _ := users.Get(ctx, &sqlpb.SqlUsersGetRequest{Project: project, Instance: "db-1", Name: "alice"})
	if len(u2.GetDatabaseRoles()) != 2 {
		t.Fatalf("user roles not updated: %+v", u2)
	}
	// User on a missing instance is NotFound.
	if _, err := users.Insert(ctx, &sqlpb.SqlUsersInsertRequest{
		Project: project, Instance: "nope", Body: &sqlpb.User{Name: "x"},
	}); status.Code(err) != codes.NotFound {
		t.Fatalf("user on missing instance: got %v", err)
	}

	// Database lifecycle.
	dbs := newDBs(st)
	if _, err := dbs.Insert(ctx, &sqlpb.SqlDatabasesInsertRequest{
		Project:  project,
		Instance: "db-1",
		Body:     &sqlpb.Database{Name: "appdb", Charset: "utf8mb4"},
	}); err != nil {
		t.Fatalf("Insert database: %v", err)
	}
	db, err := dbs.Get(ctx, &sqlpb.SqlDatabasesGetRequest{Project: project, Instance: "db-1", Database: "appdb"})
	if err != nil || db.GetCharset() != "utf8mb4" || db.GetKind() != "sql#database" || db.GetEtag() == "" {
		t.Fatalf("Get database: %v %+v", err, db)
	}
	dl, err := dbs.List(ctx, &sqlpb.SqlDatabasesListRequest{Project: project, Instance: "db-1"})
	if err != nil || len(dl.Items) != 1 || dl.Kind != "sql#databasesList" {
		t.Fatalf("List databases: %v %+v", err, dl)
	}
	if _, err := dbs.Update(ctx, &sqlpb.SqlDatabasesUpdateRequest{
		Project: project, Instance: "db-1", Database: "appdb",
		Body: &sqlpb.Database{Collation: "utf8mb4_bin"},
	}); err != nil {
		t.Fatalf("Update database: %v", err)
	}
	db2, _ := dbs.Get(ctx, &sqlpb.SqlDatabasesGetRequest{Project: project, Instance: "db-1", Database: "appdb"})
	if db2.GetCollation() != "utf8mb4_bin" || db2.GetName() != "appdb" {
		t.Fatalf("database update: %+v", db2)
	}

	// Backup run lifecycle.
	brs := newBackupRuns(st)
	bop, err := brs.Insert(ctx, &sqlpb.SqlBackupRunsInsertRequest{Project: project, Instance: "db-1"})
	if err != nil || bop.GetOperationType() != sqlpb.Operation_BACKUP {
		t.Fatalf("Insert backup run: %v %+v", err, bop)
	}
	bl, err := brs.List(ctx, &sqlpb.SqlBackupRunsListRequest{Project: project, Instance: "db-1"})
	if err != nil || len(bl.Items) != 1 {
		t.Fatalf("List backup runs: %v %+v", err, bl)
	}
	br := bl.Items[0]
	if br.GetStatus() != sqlpb.SqlBackupRunStatus_SUCCESSFUL || br.GetType() != sqlpb.SqlBackupRunType_ON_DEMAND || br.GetId() == 0 {
		t.Fatalf("unexpected backup run: %+v", br)
	}
	bg, err := brs.Get(ctx, &sqlpb.SqlBackupRunsGetRequest{Project: project, Instance: "db-1", Id: br.GetId()})
	if err != nil || bg.GetId() != br.GetId() {
		t.Fatalf("Get backup run: %v %+v", err, bg)
	}
	if _, err := brs.Delete(ctx, &sqlpb.SqlBackupRunsDeleteRequest{Project: project, Instance: "db-1", Id: br.GetId()}); err != nil {
		t.Fatalf("Delete backup run: %v", err)
	}
	if _, err := brs.Get(ctx, &sqlpb.SqlBackupRunsGetRequest{Project: project, Instance: "db-1", Id: br.GetId()}); status.Code(err) != codes.NotFound {
		t.Fatalf("get deleted backup run: got %v", err)
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

	ic := sqlpb.NewSqlInstancesServiceClient(conn)
	if _, err := ic.Insert(context.Background(), &sqlpb.SqlInstancesInsertRequest{
		Project: project,
		Body:    &sqlpb.DatabaseInstance{Name: "grpc-db", Region: "us-east1"},
	}); err != nil {
		t.Fatalf("gRPC Insert: %v", err)
	}
	got, err := ic.Get(context.Background(), &sqlpb.SqlInstancesGetRequest{Project: project, Instance: "grpc-db"})
	if err != nil || got.GetName() != "grpc-db" {
		t.Fatalf("gRPC Get: %v %+v", err, got)
	}
	oc := sqlpb.NewSqlOperationsServiceClient(conn)
	ops, err := oc.List(context.Background(), &sqlpb.SqlOperationsListRequest{Project: project})
	if err != nil || len(ops.Items) != 1 {
		t.Fatalf("gRPC List operations: %v %+v", err, ops)
	}
	// A scaffolded SQL service is not registered here at all; an unknown
	// method on a registered service stays UNIMPLEMENTED.
	if _, err := oc.Cancel(context.Background(), &sqlpb.SqlOperationsCancelRequest{Project: project, Operation: "1"}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("gRPC Cancel: got %v, want Unimplemented", err)
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

	// Insert instance via REST (body is the DatabaseInstance).
	req, _ := http.NewRequest("POST", ts.URL+"/v1/projects/"+project+"/instances", strings.NewReader(
		`{"name":"rest-db","region":"us-central1","databaseVersion":"MYSQL_8_0"}`))
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
	resp2, err := http.Get(ts.URL + "/v1/projects/" + project + "/instances/rest-db")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var inst map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&inst); err != nil {
		t.Fatal(err)
	}
	if inst["name"] != "rest-db" || inst["state"] != "RUNNABLE" || inst["region"] != "us-central1" {
		t.Fatalf("unexpected instance: %v", inst)
	}

	// Insert user via REST.
	req3, _ := http.NewRequest("POST", ts.URL+"/v1/projects/"+project+"/instances/rest-db/users", strings.NewReader(`{"name":"bob","password":"pw"}`))
	req3.Header.Set("Content-Type", "application/json")
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != 200 {
		raw, _ := io.ReadAll(resp3.Body)
		t.Fatalf("insert user status = %d body=%s", resp3.StatusCode, raw)
	}

	// List databases (empty) is well-formed.
	resp4, err := http.Get(ts.URL + "/v1/projects/" + project + "/instances/rest-db/databases")
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	var dl map[string]any
	if err := json.NewDecoder(resp4.Body).Decode(&dl); err != nil {
		t.Fatal(err)
	}
	if dl["kind"] != "sql#databasesList" {
		t.Fatalf("unexpected databases list: %v", dl)
	}

	// Operations list via REST.
	resp5, err := http.Get(ts.URL + "/v1/projects/" + project + "/operations")
	if err != nil {
		t.Fatal(err)
	}
	defer resp5.Body.Close()
	var ol map[string]any
	if err := json.NewDecoder(resp5.Body).Decode(&ol); err != nil {
		t.Fatal(err)
	}
	if ol["kind"] != "sql#operationsList" {
		t.Fatalf("unexpected operations list: %v", ol)
	}

	// Delete instance.
	req6, _ := http.NewRequest("DELETE", ts.URL+"/v1/projects/"+project+"/instances/rest-db", nil)
	resp6, err := http.DefaultClient.Do(req6)
	if err != nil {
		t.Fatal(err)
	}
	defer resp6.Body.Close()
	if resp6.StatusCode != 200 {
		raw, _ := io.ReadAll(resp6.Body)
		t.Fatalf("delete status = %d body=%s", resp6.StatusCode, raw)
	}
}

func TestSslCertificatesAndAdvancedInstanceOperations(t *testing.T) {
	ctx := context.Background()
	st := NewState()
	instances := &instancesServer{state: st}
	if _, err := instances.Insert(ctx, &sqlpb.SqlInstancesInsertRequest{Project: project, Body: &sqlpb.DatabaseInstance{Name: "ssl-db"}}); err != nil {
		t.Fatalf("insert instance: %v", err)
	}

	certs := &sslCertsServer{state: st}
	ins, err := certs.Insert(ctx, &sqlpb.SqlSslCertsInsertRequest{
		Project: project, Instance: "ssl-db",
		Body: &sqlpb.SslCertsInsertRequest{CommonName: "client-1"},
	})
	if err != nil {
		t.Fatalf("insert ssl cert: %v", err)
	}
	cert := ins.GetClientCert().GetCertInfo()
	if cert.GetCert() == "" || ins.GetClientCert().GetCertPrivateKey() == "" || ins.GetServerCaCert().GetCert() == "" {
		t.Fatalf("incomplete cert response: %+v", ins)
	}
	got, err := certs.Get(ctx, &sqlpb.SqlSslCertsGetRequest{Project: project, Instance: "ssl-db", Sha1Fingerprint: cert.GetSha1Fingerprint()})
	if err != nil || got.GetCommonName() != "client-1" {
		t.Fatalf("get ssl cert: %v %+v", err, got)
	}
	list, err := certs.List(ctx, &sqlpb.SqlSslCertsListRequest{Project: project, Instance: "ssl-db"})
	if err != nil || len(list.GetItems()) != 1 {
		t.Fatalf("list ssl certs: %v %+v", err, list)
	}
	eph, err := instances.CreateEphemeral(ctx, &sqlpb.SqlInstancesCreateEphemeralCertRequest{Project: project, Instance: "ssl-db", Body: &sqlpb.SslCertsCreateEphemeralRequest{PublicKey: "unused"}})
	if err != nil || eph.GetCert() == "" {
		t.Fatalf("create ephemeral cert: %v %+v", err, eph)
	}
	if _, err := instances.ResetSslConfig(ctx, &sqlpb.SqlInstancesResetSslConfigRequest{Project: project, Instance: "ssl-db"}); err != nil {
		t.Fatalf("reset ssl config: %v", err)
	}
	list, err = certs.List(ctx, &sqlpb.SqlSslCertsListRequest{Project: project, Instance: "ssl-db"})
	if err != nil || len(list.GetItems()) != 0 {
		t.Fatalf("ssl certs after reset: %v %+v", err, list)
	}

	if _, err := instances.Stop(ctx, &sqlpb.SqlInstancesStopRequest{Project: project, Instance: "ssl-db"}); err != nil {
		t.Fatalf("stop instance: %v", err)
	}
	inst, _ := instances.Get(ctx, &sqlpb.SqlInstancesGetRequest{Project: project, Instance: "ssl-db"})
	if inst.GetState() != sqlpb.DatabaseInstance_SUSPENDED {
		t.Fatalf("state after stop: %v", inst.GetState())
	}
	if _, err := instances.Start(ctx, &sqlpb.SqlInstancesStartRequest{Project: project, Instance: "ssl-db"}); err != nil {
		t.Fatalf("start instance: %v", err)
	}
	if _, err := instances.Restart(ctx, &sqlpb.SqlInstancesRestartRequest{Project: project, Instance: "ssl-db"}); err != nil {
		t.Fatalf("restart instance: %v", err)
	}
	if _, err := instances.Clone(ctx, &sqlpb.SqlInstancesCloneRequest{Project: project, Instance: "ssl-db", Body: &sqlpb.InstancesCloneRequest{CloneContext: &sqlpb.CloneContext{DestinationInstanceName: "ssl-db-clone"}}}); err != nil {
		t.Fatalf("clone instance: %v", err)
	}
	clone, err := instances.Get(ctx, &sqlpb.SqlInstancesGetRequest{Project: project, Instance: "ssl-db-clone"})
	if err != nil || clone.GetName() != "ssl-db-clone" {
		t.Fatalf("get clone: %v %+v", err, clone)
	}

	customerCfg := &sqlpb.CustomerManagedCertificateConfig{}
	if _, err := certs.UpdateCustomerManagedCertificate(ctx, &sqlpb.UpdateCustomerManagedCertificateRequest{Project: project, Instance: "ssl-db", CertificateBundle: "bundle1", AllowMissing: true, CustomerManagedCertificateConfig: customerCfg}); err != nil {
		t.Fatalf("update customer certificate: %v", err)
	}
	customerList, err := certs.ListCustomerManagedCertificates(ctx, &sqlpb.ListCustomerManagedCertificatesRequest{Project: project, Instance: "ssl-db"})
	if err != nil || len(customerList.GetCustomerManagedCertificates()) != 1 {
		t.Fatalf("list customer certificates: %v %+v", err, customerList)
	}
	if _, err := certs.DeleteCustomerManagedCertificate(ctx, &sqlpb.DeleteCustomerManagedCertificateRequest{Project: project, Instance: "ssl-db", CertificateBundle: "bundle1"}); err != nil {
		t.Fatalf("delete customer certificate: %v", err)
	}
}
