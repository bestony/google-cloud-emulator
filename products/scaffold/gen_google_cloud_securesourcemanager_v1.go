package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/securesourcemanager/v1"
)

func init() {
	registerEntry("google.cloud.securesourcemanager.v1.SecureSourceManager",
		func(s grpc.ServiceRegistrar) { pb.RegisterSecureSourceManagerServer(s, &pb.UnimplementedSecureSourceManagerServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSecureSourceManagerHandlerServer(ctx, mux, &pb.UnimplementedSecureSourceManagerServer{}) },
	)
}
