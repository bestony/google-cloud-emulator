package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/apptopology/v1"
)

func init() {
	registerEntry("google.cloud.apptopology.v1.AppTopology",
		func(s grpc.ServiceRegistrar) { pb.RegisterAppTopologyServer(s, &pb.UnimplementedAppTopologyServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAppTopologyHandlerServer(ctx, mux, &pb.UnimplementedAppTopologyServer{}) },
	)
}
