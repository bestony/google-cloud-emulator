package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/memorystore/v1"
)

func init() {
	registerEntry("google.cloud.memorystore.v1.Memorystore",
		func(s grpc.ServiceRegistrar) { pb.RegisterMemorystoreServer(s, &pb.UnimplementedMemorystoreServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterMemorystoreHandlerServer(ctx, mux, &pb.UnimplementedMemorystoreServer{}) },
	)
}
