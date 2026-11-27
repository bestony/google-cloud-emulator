package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/securityposture/v1"
)

func init() {
	registerEntry("google.cloud.securityposture.v1.SecurityPosture",
		func(s grpc.ServiceRegistrar) { pb.RegisterSecurityPostureServer(s, &pb.UnimplementedSecurityPostureServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSecurityPostureHandlerServer(ctx, mux, &pb.UnimplementedSecurityPostureServer{}) },
	)
}
