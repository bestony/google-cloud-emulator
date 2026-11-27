package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/workloadmanager/v1"
)

func init() {
	registerEntry("google.cloud.workloadmanager.v1.WorkloadManager",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterWorkloadManagerServer(s, &pb.UnimplementedWorkloadManagerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterWorkloadManagerHandlerServer(ctx, mux, &pb.UnimplementedWorkloadManagerServer{})
		},
	)
}
