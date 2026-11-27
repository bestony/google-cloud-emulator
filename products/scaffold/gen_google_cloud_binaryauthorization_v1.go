package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/binaryauthorization/v1"
)

func init() {
	registerEntry("google.cloud.binaryauthorization.v1.BinauthzManagementServiceV1",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterBinauthzManagementServiceV1Server(s, &pb.UnimplementedBinauthzManagementServiceV1Server{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterBinauthzManagementServiceV1HandlerServer(ctx, mux, &pb.UnimplementedBinauthzManagementServiceV1Server{})
		},
	)
	registerEntry("google.cloud.binaryauthorization.v1.SystemPolicyV1",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSystemPolicyV1Server(s, &pb.UnimplementedSystemPolicyV1Server{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSystemPolicyV1HandlerServer(ctx, mux, &pb.UnimplementedSystemPolicyV1Server{})
		},
	)
	registerEntry("google.cloud.binaryauthorization.v1.ValidationHelperV1",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterValidationHelperV1Server(s, &pb.UnimplementedValidationHelperV1Server{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterValidationHelperV1HandlerServer(ctx, mux, &pb.UnimplementedValidationHelperV1Server{})
		},
	)
}
