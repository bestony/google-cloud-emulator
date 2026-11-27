package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/productregistry/v1"
)

func init() {
	registerEntry("google.cloud.productregistry.v1.CloudProductRegistryReadService",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudProductRegistryReadServiceServer(s, &pb.UnimplementedCloudProductRegistryReadServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudProductRegistryReadServiceHandlerServer(ctx, mux, &pb.UnimplementedCloudProductRegistryReadServiceServer{}) },
	)
}
