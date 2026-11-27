package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/gkemulticloud/v1"
)

func init() {
	registerEntry("google.cloud.gkemulticloud.v1.AwsClusters",
		func(s grpc.ServiceRegistrar) { pb.RegisterAwsClustersServer(s, &pb.UnimplementedAwsClustersServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAwsClustersHandlerServer(ctx, mux, &pb.UnimplementedAwsClustersServer{}) },
	)
	registerEntry("google.cloud.gkemulticloud.v1.AzureClusters",
		func(s grpc.ServiceRegistrar) { pb.RegisterAzureClustersServer(s, &pb.UnimplementedAzureClustersServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAzureClustersHandlerServer(ctx, mux, &pb.UnimplementedAzureClustersServer{}) },
	)
	registerEntry("google.cloud.gkemulticloud.v1.AttachedClusters",
		func(s grpc.ServiceRegistrar) { pb.RegisterAttachedClustersServer(s, &pb.UnimplementedAttachedClustersServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAttachedClustersHandlerServer(ctx, mux, &pb.UnimplementedAttachedClustersServer{}) },
	)
}
