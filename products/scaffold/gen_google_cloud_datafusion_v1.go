package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/datafusion/v1"
)

func init() {
	registerEntry("google.cloud.datafusion.v1.DataFusion",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataFusionServer(s, &pb.UnimplementedDataFusionServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDataFusionHandlerServer(ctx, mux, &pb.UnimplementedDataFusionServer{}) },
	)
}
