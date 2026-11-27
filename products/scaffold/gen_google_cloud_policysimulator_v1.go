package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/policysimulator/v1"
)

func init() {
	registerEntry("google.cloud.policysimulator.v1.Simulator",
		func(s grpc.ServiceRegistrar) { pb.RegisterSimulatorServer(s, &pb.UnimplementedSimulatorServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSimulatorHandlerServer(ctx, mux, &pb.UnimplementedSimulatorServer{})
		},
	)
	registerEntry("google.cloud.policysimulator.v1.OrgPolicyViolationsPreviewService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterOrgPolicyViolationsPreviewServiceServer(s, &pb.UnimplementedOrgPolicyViolationsPreviewServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterOrgPolicyViolationsPreviewServiceHandlerServer(ctx, mux, &pb.UnimplementedOrgPolicyViolationsPreviewServiceServer{})
		},
	)
}
