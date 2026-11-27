package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/edgecontainer/v1"
)

func init() {
	registerEntry("google.cloud.edgecontainer.v1.EdgeContainer",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterEdgeContainerServer(s, &pb.UnimplementedEdgeContainerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterEdgeContainerHandlerServer(ctx, mux, &pb.UnimplementedEdgeContainerServer{})
		},
	)
}
