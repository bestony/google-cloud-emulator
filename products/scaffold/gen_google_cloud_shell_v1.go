package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/shell/v1"
)

func init() {
	registerEntry("google.cloud.shell.v1.CloudShellService",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudShellServiceServer(s, &pb.UnimplementedCloudShellServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudShellServiceHandlerServer(ctx, mux, &pb.UnimplementedCloudShellServiceServer{}) },
	)
}
