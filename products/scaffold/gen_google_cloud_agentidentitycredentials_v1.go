package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/agentidentitycredentials/v1"
)

func init() {
	registerEntry("google.cloud.agentidentitycredentials.v1.AuthProviderCredentialsService",
		func(s grpc.ServiceRegistrar) { pb.RegisterAuthProviderCredentialsServiceServer(s, &pb.UnimplementedAuthProviderCredentialsServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAuthProviderCredentialsServiceHandlerServer(ctx, mux, &pb.UnimplementedAuthProviderCredentialsServiceServer{}) },
	)
}
