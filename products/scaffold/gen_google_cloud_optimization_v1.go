package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/optimization/v1"
)

func init() {
	registerEntry("google.cloud.optimization.v1.FleetRouting",
		func(s grpc.ServiceRegistrar) { pb.RegisterFleetRoutingServer(s, &pb.UnimplementedFleetRoutingServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterFleetRoutingHandlerServer(ctx, mux, &pb.UnimplementedFleetRoutingServer{})
		},
	)
}
