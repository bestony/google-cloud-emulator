package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/lustre/v1"
)

func init() {
	registerEntry("google.cloud.lustre.v1.Lustre",
		func(s grpc.ServiceRegistrar) { pb.RegisterLustreServer(s, &pb.UnimplementedLustreServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterLustreHandlerServer(ctx, mux, &pb.UnimplementedLustreServer{}) },
	)
}
