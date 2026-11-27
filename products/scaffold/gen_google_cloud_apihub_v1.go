package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/apihub/v1"
)

func init() {
	registerEntry("google.cloud.apihub.v1.ApiHubCurate",
		func(s grpc.ServiceRegistrar) { pb.RegisterApiHubCurateServer(s, &pb.UnimplementedApiHubCurateServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterApiHubCurateHandlerServer(ctx, mux, &pb.UnimplementedApiHubCurateServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.HostProjectRegistrationService",
		func(s grpc.ServiceRegistrar) { pb.RegisterHostProjectRegistrationServiceServer(s, &pb.UnimplementedHostProjectRegistrationServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterHostProjectRegistrationServiceHandlerServer(ctx, mux, &pb.UnimplementedHostProjectRegistrationServiceServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.ApiHubDiscovery",
		func(s grpc.ServiceRegistrar) { pb.RegisterApiHubDiscoveryServer(s, &pb.UnimplementedApiHubDiscoveryServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterApiHubDiscoveryHandlerServer(ctx, mux, &pb.UnimplementedApiHubDiscoveryServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.ApiHubCollect",
		func(s grpc.ServiceRegistrar) { pb.RegisterApiHubCollectServer(s, &pb.UnimplementedApiHubCollectServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterApiHubCollectHandlerServer(ctx, mux, &pb.UnimplementedApiHubCollectServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.ApiHubPlugin",
		func(s grpc.ServiceRegistrar) { pb.RegisterApiHubPluginServer(s, &pb.UnimplementedApiHubPluginServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterApiHubPluginHandlerServer(ctx, mux, &pb.UnimplementedApiHubPluginServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.RuntimeProjectAttachmentService",
		func(s grpc.ServiceRegistrar) { pb.RegisterRuntimeProjectAttachmentServiceServer(s, &pb.UnimplementedRuntimeProjectAttachmentServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRuntimeProjectAttachmentServiceHandlerServer(ctx, mux, &pb.UnimplementedRuntimeProjectAttachmentServiceServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.ApiHubAddon",
		func(s grpc.ServiceRegistrar) { pb.RegisterApiHubAddonServer(s, &pb.UnimplementedApiHubAddonServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterApiHubAddonHandlerServer(ctx, mux, &pb.UnimplementedApiHubAddonServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.LintingService",
		func(s grpc.ServiceRegistrar) { pb.RegisterLintingServiceServer(s, &pb.UnimplementedLintingServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterLintingServiceHandlerServer(ctx, mux, &pb.UnimplementedLintingServiceServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.Provisioning",
		func(s grpc.ServiceRegistrar) { pb.RegisterProvisioningServer(s, &pb.UnimplementedProvisioningServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterProvisioningHandlerServer(ctx, mux, &pb.UnimplementedProvisioningServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.ApiHub",
		func(s grpc.ServiceRegistrar) { pb.RegisterApiHubServer(s, &pb.UnimplementedApiHubServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterApiHubHandlerServer(ctx, mux, &pb.UnimplementedApiHubServer{}) },
	)
	registerEntry("google.cloud.apihub.v1.ApiHubDependencies",
		func(s grpc.ServiceRegistrar) { pb.RegisterApiHubDependenciesServer(s, &pb.UnimplementedApiHubDependenciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterApiHubDependenciesHandlerServer(ctx, mux, &pb.UnimplementedApiHubDependenciesServer{}) },
	)
}
