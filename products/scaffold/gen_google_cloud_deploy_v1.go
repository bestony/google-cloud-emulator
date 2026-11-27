package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/deploy/v1"
)

func init() {
	registerEntry("google.cloud.deploy.v1.CloudDeploy",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudDeployServer(s, &pb.UnimplementedCloudDeployServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudDeployHandlerServer(ctx, mux, &pb.UnimplementedCloudDeployServer{}) },
	)
}
