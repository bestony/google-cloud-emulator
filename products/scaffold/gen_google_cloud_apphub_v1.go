package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/apphub/v1"
)

func init() {
	registerEntry("google.cloud.apphub.v1.AppHub",
		func(s grpc.ServiceRegistrar) { pb.RegisterAppHubServer(s, &pb.UnimplementedAppHubServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAppHubHandlerServer(ctx, mux, &pb.UnimplementedAppHubServer{}) },
	)
}
