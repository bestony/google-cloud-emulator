package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/assuredworkloads/v1"
)

func init() {
	registerEntry("google.cloud.assuredworkloads.v1.AssuredWorkloadsService",
		func(s grpc.ServiceRegistrar) { pb.RegisterAssuredWorkloadsServiceServer(s, &pb.UnimplementedAssuredWorkloadsServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAssuredWorkloadsServiceHandlerServer(ctx, mux, &pb.UnimplementedAssuredWorkloadsServiceServer{}) },
	)
}
