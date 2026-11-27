package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/clouddms/v1"
)

func init() {
	registerEntry("google.cloud.clouddms.v1.DataMigrationService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataMigrationServiceServer(s, &pb.UnimplementedDataMigrationServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataMigrationServiceHandlerServer(ctx, mux, &pb.UnimplementedDataMigrationServiceServer{})
		},
	)
}
