package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/networkconnectivity/v1"
)

func init() {
	registerEntry("google.cloud.networkconnectivity.v1.PolicyBasedRoutingService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterPolicyBasedRoutingServiceServer(s, &pb.UnimplementedPolicyBasedRoutingServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterPolicyBasedRoutingServiceHandlerServer(ctx, mux, &pb.UnimplementedPolicyBasedRoutingServiceServer{})
		},
	)
	registerEntry("google.cloud.networkconnectivity.v1.CrossNetworkAutomationService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCrossNetworkAutomationServiceServer(s, &pb.UnimplementedCrossNetworkAutomationServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCrossNetworkAutomationServiceHandlerServer(ctx, mux, &pb.UnimplementedCrossNetworkAutomationServiceServer{})
		},
	)
	registerEntry("google.cloud.networkconnectivity.v1.DataTransferService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataTransferServiceServer(s, &pb.UnimplementedDataTransferServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataTransferServiceHandlerServer(ctx, mux, &pb.UnimplementedDataTransferServiceServer{})
		},
	)
	registerEntry("google.cloud.networkconnectivity.v1.HubService",
		func(s grpc.ServiceRegistrar) { pb.RegisterHubServiceServer(s, &pb.UnimplementedHubServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterHubServiceHandlerServer(ctx, mux, &pb.UnimplementedHubServiceServer{})
		},
	)
	registerEntry("google.cloud.networkconnectivity.v1.InternalRangeService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterInternalRangeServiceServer(s, &pb.UnimplementedInternalRangeServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterInternalRangeServiceHandlerServer(ctx, mux, &pb.UnimplementedInternalRangeServiceServer{})
		},
	)
}
