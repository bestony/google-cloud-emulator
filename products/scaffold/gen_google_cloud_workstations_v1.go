package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/workstations/v1"
)

func init() {
	registerEntry("google.cloud.workstations.v1.Workstations",
		func(s grpc.ServiceRegistrar) { pb.RegisterWorkstationsServer(s, &pb.UnimplementedWorkstationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterWorkstationsHandlerServer(ctx, mux, &pb.UnimplementedWorkstationsServer{}) },
	)
}
