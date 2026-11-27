package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/securitycentermanagement/v1"
)

func init() {
	registerEntry("google.cloud.securitycentermanagement.v1.SecurityCenterManagement",
		func(s grpc.ServiceRegistrar) { pb.RegisterSecurityCenterManagementServer(s, &pb.UnimplementedSecurityCenterManagementServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSecurityCenterManagementHandlerServer(ctx, mux, &pb.UnimplementedSecurityCenterManagementServer{}) },
	)
}
