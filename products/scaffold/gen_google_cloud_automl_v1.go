package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/automl/v1"
)

func init() {
	registerEntry("google.cloud.automl.v1.AutoMl",
		func(s grpc.ServiceRegistrar) { pb.RegisterAutoMlServer(s, &pb.UnimplementedAutoMlServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAutoMlHandlerServer(ctx, mux, &pb.UnimplementedAutoMlServer{})
		},
	)
	registerEntry("google.cloud.automl.v1.PredictionService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterPredictionServiceServer(s, &pb.UnimplementedPredictionServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterPredictionServiceHandlerServer(ctx, mux, &pb.UnimplementedPredictionServiceServer{})
		},
	)
}
