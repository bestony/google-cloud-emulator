package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/migrationcenter/v1"
)

func init() {
	registerEntry("google.cloud.migrationcenter.v1.MigrationCenter",
		func(s grpc.ServiceRegistrar) { pb.RegisterMigrationCenterServer(s, &pb.UnimplementedMigrationCenterServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterMigrationCenterHandlerServer(ctx, mux, &pb.UnimplementedMigrationCenterServer{}) },
	)
}
