package sql

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	sqlpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/sql/v1"
)

// Register wires the implemented Cloud SQL services into the gRPC server
// and REST gateway. All other sql.v1 services remain scaffolded
// UNIMPLEMENTED.
func Register(grpcServer *grpc.Server, gateway *runtime.ServeMux, state *State) error {
	ctx := context.Background()

	instances := &instancesServer{state: state}
	operations := &operationsServer{state: state}
	users := &usersServer{state: state}
	databases := &databasesServer{state: state}
	backupRuns := &backupRunsServer{state: state}
	flags := &flagsServer{}
	sslCerts := &sslCertsServer{state: state}

	sqlpb.RegisterSqlInstancesServiceServer(grpcServer, instances)
	sqlpb.RegisterSqlOperationsServiceServer(grpcServer, operations)
	sqlpb.RegisterSqlUsersServiceServer(grpcServer, users)
	sqlpb.RegisterSqlDatabasesServiceServer(grpcServer, databases)
	sqlpb.RegisterSqlBackupRunsServiceServer(grpcServer, backupRuns)
	sqlpb.RegisterSqlFlagsServiceServer(grpcServer, flags)
	sqlpb.RegisterSqlSslCertsServiceServer(grpcServer, sslCerts)

	if err := sqlpb.RegisterSqlInstancesServiceHandlerServer(ctx, gateway, instances); err != nil {
		return err
	}
	if err := sqlpb.RegisterSqlOperationsServiceHandlerServer(ctx, gateway, operations); err != nil {
		return err
	}
	if err := sqlpb.RegisterSqlUsersServiceHandlerServer(ctx, gateway, users); err != nil {
		return err
	}
	if err := sqlpb.RegisterSqlDatabasesServiceHandlerServer(ctx, gateway, databases); err != nil {
		return err
	}
	if err := sqlpb.RegisterSqlBackupRunsServiceHandlerServer(ctx, gateway, backupRuns); err != nil {
		return err
	}
	if err := sqlpb.RegisterSqlFlagsServiceHandlerServer(ctx, gateway, flags); err != nil {
		return err
	}
	if err := sqlpb.RegisterSqlSslCertsServiceHandlerServer(ctx, gateway, sslCerts); err != nil {
		return err
	}
	return nil
}
