package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/webrisk/v1"
)

func init() {
	registerEntry("google.cloud.webrisk.v1.WebRiskService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterWebRiskServiceServer(s, &pb.UnimplementedWebRiskServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterWebRiskServiceHandlerServer(ctx, mux, &pb.UnimplementedWebRiskServiceServer{})
		},
	)
}
