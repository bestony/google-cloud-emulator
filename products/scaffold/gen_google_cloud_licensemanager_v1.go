package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/licensemanager/v1"
)

func init() {
	registerEntry("google.cloud.licensemanager.v1.LicenseManager",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterLicenseManagerServer(s, &pb.UnimplementedLicenseManagerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterLicenseManagerHandlerServer(ctx, mux, &pb.UnimplementedLicenseManagerServer{})
		},
	)
}
