package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/workflows/v1"
)

func init() {
	registerEntry("google.cloud.workflows.v1.Workflows",
		func(s grpc.ServiceRegistrar) { pb.RegisterWorkflowsServer(s, &pb.UnimplementedWorkflowsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterWorkflowsHandlerServer(ctx, mux, &pb.UnimplementedWorkflowsServer{})
		},
	)
}
