package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/edgenetwork/v1"
)

func init() {
	registerEntry("google.cloud.edgenetwork.v1.EdgeNetwork",
		func(s grpc.ServiceRegistrar) { pb.RegisterEdgeNetworkServer(s, &pb.UnimplementedEdgeNetworkServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterEdgeNetworkHandlerServer(ctx, mux, &pb.UnimplementedEdgeNetworkServer{}) },
	)
}
