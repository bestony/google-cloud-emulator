package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/agentidentity/v1"
)

func init() {
	registerEntry("google.cloud.agentidentity.v1.AuthProviderService",
		func(s grpc.ServiceRegistrar) { pb.RegisterAuthProviderServiceServer(s, &pb.UnimplementedAuthProviderServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAuthProviderServiceHandlerServer(ctx, mux, &pb.UnimplementedAuthProviderServiceServer{}) },
	)
}
