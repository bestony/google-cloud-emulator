package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/apiregistry/v1"
)

func init() {
	registerEntry("google.cloud.apiregistry.v1.CloudApiRegistry",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudApiRegistryServer(s, &pb.UnimplementedCloudApiRegistryServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudApiRegistryHandlerServer(ctx, mux, &pb.UnimplementedCloudApiRegistryServer{}) },
	)
}
