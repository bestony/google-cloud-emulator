package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/cloudcontrolspartner/v1"
)

func init() {
	registerEntry("google.cloud.cloudcontrolspartner.v1.CloudControlsPartnerCore",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudControlsPartnerCoreServer(s, &pb.UnimplementedCloudControlsPartnerCoreServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudControlsPartnerCoreHandlerServer(ctx, mux, &pb.UnimplementedCloudControlsPartnerCoreServer{}) },
	)
	registerEntry("google.cloud.cloudcontrolspartner.v1.CloudControlsPartnerMonitoring",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudControlsPartnerMonitoringServer(s, &pb.UnimplementedCloudControlsPartnerMonitoringServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudControlsPartnerMonitoringHandlerServer(ctx, mux, &pb.UnimplementedCloudControlsPartnerMonitoringServer{}) },
	)
}
