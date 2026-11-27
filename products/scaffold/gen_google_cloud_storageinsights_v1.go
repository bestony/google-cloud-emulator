package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/storageinsights/v1"
)

func init() {
	registerEntry("google.cloud.storageinsights.v1.StorageInsights",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterStorageInsightsServer(s, &pb.UnimplementedStorageInsightsServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterStorageInsightsHandlerServer(ctx, mux, &pb.UnimplementedStorageInsightsServer{})
		},
	)
}
