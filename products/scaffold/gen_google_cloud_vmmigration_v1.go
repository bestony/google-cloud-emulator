package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/vmmigration/v1"
)

func init() {
	registerEntry("google.cloud.vmmigration.v1.VmMigration",
		func(s grpc.ServiceRegistrar) { pb.RegisterVmMigrationServer(s, &pb.UnimplementedVmMigrationServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterVmMigrationHandlerServer(ctx, mux, &pb.UnimplementedVmMigrationServer{})
		},
	)
}
