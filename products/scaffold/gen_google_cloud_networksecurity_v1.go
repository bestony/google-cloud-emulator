package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/networksecurity/v1"
)

func init() {
	registerEntry("google.cloud.networksecurity.v1.AddressGroupService",
		func(s grpc.ServiceRegistrar) { pb.RegisterAddressGroupServiceServer(s, &pb.UnimplementedAddressGroupServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAddressGroupServiceHandlerServer(ctx, mux, &pb.UnimplementedAddressGroupServiceServer{}) },
	)
	registerEntry("google.cloud.networksecurity.v1.OrganizationAddressGroupService",
		func(s grpc.ServiceRegistrar) { pb.RegisterOrganizationAddressGroupServiceServer(s, &pb.UnimplementedOrganizationAddressGroupServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterOrganizationAddressGroupServiceHandlerServer(ctx, mux, &pb.UnimplementedOrganizationAddressGroupServiceServer{}) },
	)
	registerEntry("google.cloud.networksecurity.v1.Intercept",
		func(s grpc.ServiceRegistrar) { pb.RegisterInterceptServer(s, &pb.UnimplementedInterceptServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInterceptHandlerServer(ctx, mux, &pb.UnimplementedInterceptServer{}) },
	)
	registerEntry("google.cloud.networksecurity.v1.SecurityProfileGroupService",
		func(s grpc.ServiceRegistrar) { pb.RegisterSecurityProfileGroupServiceServer(s, &pb.UnimplementedSecurityProfileGroupServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSecurityProfileGroupServiceHandlerServer(ctx, mux, &pb.UnimplementedSecurityProfileGroupServiceServer{}) },
	)
	registerEntry("google.cloud.networksecurity.v1.OrganizationSecurityProfileGroupService",
		func(s grpc.ServiceRegistrar) { pb.RegisterOrganizationSecurityProfileGroupServiceServer(s, &pb.UnimplementedOrganizationSecurityProfileGroupServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterOrganizationSecurityProfileGroupServiceHandlerServer(ctx, mux, &pb.UnimplementedOrganizationSecurityProfileGroupServiceServer{}) },
	)
	registerEntry("google.cloud.networksecurity.v1.NetworkSecurity",
		func(s grpc.ServiceRegistrar) { pb.RegisterNetworkSecurityServer(s, &pb.UnimplementedNetworkSecurityServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNetworkSecurityHandlerServer(ctx, mux, &pb.UnimplementedNetworkSecurityServer{}) },
	)
	registerEntry("google.cloud.networksecurity.v1.SSERealmService",
		func(s grpc.ServiceRegistrar) { pb.RegisterSSERealmServiceServer(s, &pb.UnimplementedSSERealmServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSSERealmServiceHandlerServer(ctx, mux, &pb.UnimplementedSSERealmServiceServer{}) },
	)
	registerEntry("google.cloud.networksecurity.v1.FirewallActivation",
		func(s grpc.ServiceRegistrar) { pb.RegisterFirewallActivationServer(s, &pb.UnimplementedFirewallActivationServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterFirewallActivationHandlerServer(ctx, mux, &pb.UnimplementedFirewallActivationServer{}) },
	)
	registerEntry("google.cloud.networksecurity.v1.DnsThreatDetectorService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDnsThreatDetectorServiceServer(s, &pb.UnimplementedDnsThreatDetectorServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDnsThreatDetectorServiceHandlerServer(ctx, mux, &pb.UnimplementedDnsThreatDetectorServiceServer{}) },
	)
	registerEntry("google.cloud.networksecurity.v1.Mirroring",
		func(s grpc.ServiceRegistrar) { pb.RegisterMirroringServer(s, &pb.UnimplementedMirroringServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterMirroringHandlerServer(ctx, mux, &pb.UnimplementedMirroringServer{}) },
	)
}
