package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/videointelligence/v1"
)

func init() {
	registerEntry("google.cloud.videointelligence.v1.VideoIntelligenceService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterVideoIntelligenceServiceServer(s, &pb.UnimplementedVideoIntelligenceServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterVideoIntelligenceServiceHandlerServer(ctx, mux, &pb.UnimplementedVideoIntelligenceServiceServer{})
		},
	)
}
