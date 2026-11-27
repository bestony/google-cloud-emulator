package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/iap/v1"
)

func init() {
	registerEntry("google.cloud.iap.v1.IdentityAwareProxyAdminService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterIdentityAwareProxyAdminServiceServer(s, &pb.UnimplementedIdentityAwareProxyAdminServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterIdentityAwareProxyAdminServiceHandlerServer(ctx, mux, &pb.UnimplementedIdentityAwareProxyAdminServiceServer{})
		},
	)
	registerEntry("google.cloud.iap.v1.IdentityAwareProxyOAuthService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterIdentityAwareProxyOAuthServiceServer(s, &pb.UnimplementedIdentityAwareProxyOAuthServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterIdentityAwareProxyOAuthServiceHandlerServer(ctx, mux, &pb.UnimplementedIdentityAwareProxyOAuthServiceServer{})
		},
	)
}
