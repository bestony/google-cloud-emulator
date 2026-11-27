package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/kms/v1"
)

func init() {
	registerEntry("google.cloud.kms.v1.AutokeyAdmin",
		func(s grpc.ServiceRegistrar) { pb.RegisterAutokeyAdminServer(s, &pb.UnimplementedAutokeyAdminServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAutokeyAdminHandlerServer(ctx, mux, &pb.UnimplementedAutokeyAdminServer{}) },
	)
	registerEntry("google.cloud.kms.v1.HsmManagement",
		func(s grpc.ServiceRegistrar) { pb.RegisterHsmManagementServer(s, &pb.UnimplementedHsmManagementServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterHsmManagementHandlerServer(ctx, mux, &pb.UnimplementedHsmManagementServer{}) },
	)
	registerEntry("google.cloud.kms.v1.EkmService",
		func(s grpc.ServiceRegistrar) { pb.RegisterEkmServiceServer(s, &pb.UnimplementedEkmServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterEkmServiceHandlerServer(ctx, mux, &pb.UnimplementedEkmServiceServer{}) },
	)
	registerEntry("google.cloud.kms.v1.Autokey",
		func(s grpc.ServiceRegistrar) { pb.RegisterAutokeyServer(s, &pb.UnimplementedAutokeyServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAutokeyHandlerServer(ctx, mux, &pb.UnimplementedAutokeyServer{}) },
	)
}
