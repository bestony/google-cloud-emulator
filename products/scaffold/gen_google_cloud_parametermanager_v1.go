package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/parametermanager/v1"
)

func init() {
	registerEntry("google.cloud.parametermanager.v1.ParameterManager",
		func(s grpc.ServiceRegistrar) { pb.RegisterParameterManagerServer(s, &pb.UnimplementedParameterManagerServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterParameterManagerHandlerServer(ctx, mux, &pb.UnimplementedParameterManagerServer{}) },
	)
}
