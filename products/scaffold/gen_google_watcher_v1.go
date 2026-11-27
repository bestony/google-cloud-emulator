package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/watcher/v1"
)

func init() {
	registerEntry("google.watcher.v1.Watcher",
		func(s grpc.ServiceRegistrar) { pb.RegisterWatcherServer(s, &pb.UnimplementedWatcherServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterWatcherHandlerServer(ctx, mux, &pb.UnimplementedWatcherServer{})
		},
	)
}
