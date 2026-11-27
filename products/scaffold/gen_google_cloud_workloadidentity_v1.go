package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/workloadidentity/v1"
)

func init() {
	registerEntry("google.cloud.workloadidentity.v1.WorkloadIdentity",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterWorkloadIdentityServer(s, &pb.UnimplementedWorkloadIdentityServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterWorkloadIdentityHandlerServer(ctx, mux, &pb.UnimplementedWorkloadIdentityServer{})
		},
	)
}
