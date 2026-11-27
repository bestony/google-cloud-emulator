package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/batch/v1"
)

func init() {
	registerEntry("google.cloud.batch.v1.BatchService",
		func(s grpc.ServiceRegistrar) { pb.RegisterBatchServiceServer(s, &pb.UnimplementedBatchServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterBatchServiceHandlerServer(ctx, mux, &pb.UnimplementedBatchServiceServer{}) },
	)
}
