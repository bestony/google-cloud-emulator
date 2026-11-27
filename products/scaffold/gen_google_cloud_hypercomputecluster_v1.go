package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/hypercomputecluster/v1"
)

func init() {
	registerEntry("google.cloud.hypercomputecluster.v1.HypercomputeCluster",
		func(s grpc.ServiceRegistrar) { pb.RegisterHypercomputeClusterServer(s, &pb.UnimplementedHypercomputeClusterServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterHypercomputeClusterHandlerServer(ctx, mux, &pb.UnimplementedHypercomputeClusterServer{}) },
	)
}
