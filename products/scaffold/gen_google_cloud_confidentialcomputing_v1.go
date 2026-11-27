package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/confidentialcomputing/v1"
)

func init() {
	registerEntry("google.cloud.confidentialcomputing.v1.ConfidentialComputing",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterConfidentialComputingServer(s, &pb.UnimplementedConfidentialComputingServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterConfidentialComputingHandlerServer(ctx, mux, &pb.UnimplementedConfidentialComputingServer{})
		},
	)
}
