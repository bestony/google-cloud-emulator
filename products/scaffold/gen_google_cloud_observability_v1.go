package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/observability/v1"
)

func init() {
	registerEntry("google.cloud.observability.v1.Observability",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterObservabilityServer(s, &pb.UnimplementedObservabilityServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterObservabilityHandlerServer(ctx, mux, &pb.UnimplementedObservabilityServer{})
		},
	)
}
