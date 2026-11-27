package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/asset/v1"
)

func init() {
	registerEntry("google.cloud.asset.v1.AssetService",
		func(s grpc.ServiceRegistrar) { pb.RegisterAssetServiceServer(s, &pb.UnimplementedAssetServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAssetServiceHandlerServer(ctx, mux, &pb.UnimplementedAssetServiceServer{}) },
	)
}
