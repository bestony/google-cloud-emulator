package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/websecurityscanner/v1"
)

func init() {
	registerEntry("google.cloud.websecurityscanner.v1.WebSecurityScanner",
		func(s grpc.ServiceRegistrar) { pb.RegisterWebSecurityScannerServer(s, &pb.UnimplementedWebSecurityScannerServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterWebSecurityScannerHandlerServer(ctx, mux, &pb.UnimplementedWebSecurityScannerServer{}) },
	)
}
