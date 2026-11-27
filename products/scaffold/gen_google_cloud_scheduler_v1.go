package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/scheduler/v1"
)

func init() {
	registerEntry("google.cloud.scheduler.v1.CloudScheduler",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCloudSchedulerServer(s, &pb.UnimplementedCloudSchedulerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCloudSchedulerHandlerServer(ctx, mux, &pb.UnimplementedCloudSchedulerServer{})
		},
	)
}
