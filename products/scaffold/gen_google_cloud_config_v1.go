package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/config/v1"
)

func init() {
	registerEntry("google.cloud.config.v1.Config",
		func(s grpc.ServiceRegistrar) { pb.RegisterConfigServer(s, &pb.UnimplementedConfigServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterConfigHandlerServer(ctx, mux, &pb.UnimplementedConfigServer{}) },
	)
}
