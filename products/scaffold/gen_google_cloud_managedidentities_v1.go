package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/managedidentities/v1"
)

func init() {
	registerEntry("google.cloud.managedidentities.v1.ManagedIdentitiesService",
		func(s grpc.ServiceRegistrar) { pb.RegisterManagedIdentitiesServiceServer(s, &pb.UnimplementedManagedIdentitiesServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterManagedIdentitiesServiceHandlerServer(ctx, mux, &pb.UnimplementedManagedIdentitiesServiceServer{}) },
	)
}
