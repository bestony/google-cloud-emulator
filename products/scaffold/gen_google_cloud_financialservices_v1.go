package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/financialservices/v1"
)

func init() {
	registerEntry("google.cloud.financialservices.v1.AML",
		func(s grpc.ServiceRegistrar) { pb.RegisterAMLServer(s, &pb.UnimplementedAMLServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAMLHandlerServer(ctx, mux, &pb.UnimplementedAMLServer{}) },
	)
}
