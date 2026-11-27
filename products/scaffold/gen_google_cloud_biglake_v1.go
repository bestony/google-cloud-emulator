package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/biglake/v1"
)

func init() {
	registerEntry("google.cloud.biglake.v1.IcebergCatalogService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterIcebergCatalogServiceServer(s, &pb.UnimplementedIcebergCatalogServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterIcebergCatalogServiceHandlerServer(ctx, mux, &pb.UnimplementedIcebergCatalogServiceServer{})
		},
	)
}
