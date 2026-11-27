package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/securitycenter/v1"
)

func init() {
	registerEntry("google.cloud.securitycenter.v1.SecurityCenter",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSecurityCenterServer(s, &pb.UnimplementedSecurityCenterServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSecurityCenterHandlerServer(ctx, mux, &pb.UnimplementedSecurityCenterServer{})
		},
	)
}
