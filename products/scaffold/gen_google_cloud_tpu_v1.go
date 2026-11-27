package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/tpu/v1"
)

func init() {
	registerEntry("google.cloud.tpu.v1.Tpu",
		func(s grpc.ServiceRegistrar) { pb.RegisterTpuServer(s, &pb.UnimplementedTpuServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterTpuHandlerServer(ctx, mux, &pb.UnimplementedTpuServer{})
		},
	)
}
