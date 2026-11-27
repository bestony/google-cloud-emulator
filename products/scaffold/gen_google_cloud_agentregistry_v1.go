package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/agentregistry/v1"
)

func init() {
	registerEntry("google.cloud.agentregistry.v1.AgentRegistry",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterAgentRegistryServer(s, &pb.UnimplementedAgentRegistryServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAgentRegistryHandlerServer(ctx, mux, &pb.UnimplementedAgentRegistryServer{})
		},
	)
}
