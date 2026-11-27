package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/eventarc/v1"
)

func init() {
	registerEntry("google.cloud.eventarc.v1.Eventarc",
		func(s grpc.ServiceRegistrar) { pb.RegisterEventarcServer(s, &pb.UnimplementedEventarcServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterEventarcHandlerServer(ctx, mux, &pb.UnimplementedEventarcServer{})
		},
	)
}
