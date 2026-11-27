package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/cloudsecuritycompliance/v1"
)

func init() {
	registerEntry("google.cloud.cloudsecuritycompliance.v1.Deployment",
		func(s grpc.ServiceRegistrar) { pb.RegisterDeploymentServer(s, &pb.UnimplementedDeploymentServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDeploymentHandlerServer(ctx, mux, &pb.UnimplementedDeploymentServer{})
		},
	)
	registerEntry("google.cloud.cloudsecuritycompliance.v1.CmEnrollmentService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCmEnrollmentServiceServer(s, &pb.UnimplementedCmEnrollmentServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCmEnrollmentServiceHandlerServer(ctx, mux, &pb.UnimplementedCmEnrollmentServiceServer{})
		},
	)
	registerEntry("google.cloud.cloudsecuritycompliance.v1.Monitoring",
		func(s grpc.ServiceRegistrar) { pb.RegisterMonitoringServer(s, &pb.UnimplementedMonitoringServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterMonitoringHandlerServer(ctx, mux, &pb.UnimplementedMonitoringServer{})
		},
	)
	registerEntry("google.cloud.cloudsecuritycompliance.v1.Audit",
		func(s grpc.ServiceRegistrar) { pb.RegisterAuditServer(s, &pb.UnimplementedAuditServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAuditHandlerServer(ctx, mux, &pb.UnimplementedAuditServer{})
		},
	)
	registerEntry("google.cloud.cloudsecuritycompliance.v1.Config",
		func(s grpc.ServiceRegistrar) { pb.RegisterConfigServer(s, &pb.UnimplementedConfigServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterConfigHandlerServer(ctx, mux, &pb.UnimplementedConfigServer{})
		},
	)
}
