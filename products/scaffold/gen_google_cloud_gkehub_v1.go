package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/gkehub/v1"
)

func init() {
	registerEntry("google.cloud.gkehub.v1.GkeHub",
		func(s grpc.ServiceRegistrar) { pb.RegisterGkeHubServer(s, &pb.UnimplementedGkeHubServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterGkeHubHandlerServer(ctx, mux, &pb.UnimplementedGkeHubServer{})
		},
	)
}
