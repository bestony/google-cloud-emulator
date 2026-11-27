package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/oslogin/v1"
)

func init() {
	registerEntry("google.cloud.oslogin.v1.OsLoginService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterOsLoginServiceServer(s, &pb.UnimplementedOsLoginServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterOsLoginServiceHandlerServer(ctx, mux, &pb.UnimplementedOsLoginServiceServer{})
		},
	)
}
