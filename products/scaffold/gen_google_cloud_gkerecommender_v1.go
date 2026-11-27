package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/gkerecommender/v1"
)

func init() {
	registerEntry("google.cloud.gkerecommender.v1.GkeInferenceQuickstart",
		func(s grpc.ServiceRegistrar) { pb.RegisterGkeInferenceQuickstartServer(s, &pb.UnimplementedGkeInferenceQuickstartServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGkeInferenceQuickstartHandlerServer(ctx, mux, &pb.UnimplementedGkeInferenceQuickstartServer{}) },
	)
}
