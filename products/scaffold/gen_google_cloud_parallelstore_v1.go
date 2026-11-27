package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/parallelstore/v1"
)

func init() {
	registerEntry("google.cloud.parallelstore.v1.Parallelstore",
		func(s grpc.ServiceRegistrar) { pb.RegisterParallelstoreServer(s, &pb.UnimplementedParallelstoreServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterParallelstoreHandlerServer(ctx, mux, &pb.UnimplementedParallelstoreServer{}) },
	)
}
