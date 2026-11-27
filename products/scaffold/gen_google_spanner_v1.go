package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/spanner/v1"
)

func init() {
	registerEntry("google.spanner.v1.Spanner",
		func(s grpc.ServiceRegistrar) { pb.RegisterSpannerServer(s, &pb.UnimplementedSpannerServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSpannerHandlerServer(ctx, mux, &pb.UnimplementedSpannerServer{}) },
	)
}
