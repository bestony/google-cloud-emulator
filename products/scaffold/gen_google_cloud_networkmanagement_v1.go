package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/networkmanagement/v1"
)

func init() {
	registerEntry("google.cloud.networkmanagement.v1.VpcFlowLogsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterVpcFlowLogsServiceServer(s, &pb.UnimplementedVpcFlowLogsServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterVpcFlowLogsServiceHandlerServer(ctx, mux, &pb.UnimplementedVpcFlowLogsServiceServer{})
		},
	)
	registerEntry("google.cloud.networkmanagement.v1.OrganizationVpcFlowLogsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterOrganizationVpcFlowLogsServiceServer(s, &pb.UnimplementedOrganizationVpcFlowLogsServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterOrganizationVpcFlowLogsServiceHandlerServer(ctx, mux, &pb.UnimplementedOrganizationVpcFlowLogsServiceServer{})
		},
	)
	registerEntry("google.cloud.networkmanagement.v1.ReachabilityService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterReachabilityServiceServer(s, &pb.UnimplementedReachabilityServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterReachabilityServiceHandlerServer(ctx, mux, &pb.UnimplementedReachabilityServiceServer{})
		},
	)
}
