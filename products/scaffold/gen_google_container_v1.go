package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/container/v1"
)

func init() {
	registerEntry("google.container.v1.ClusterManager",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterClusterManagerServer(s, &pb.UnimplementedClusterManagerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterClusterManagerHandlerServer(ctx, mux, &pb.UnimplementedClusterManagerServer{})
		},
	)
}
