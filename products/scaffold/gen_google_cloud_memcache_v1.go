package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/memcache/v1"
)

func init() {
	registerEntry("google.cloud.memcache.v1.CloudMemcache",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCloudMemcacheServer(s, &pb.UnimplementedCloudMemcacheServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCloudMemcacheHandlerServer(ctx, mux, &pb.UnimplementedCloudMemcacheServer{})
		},
	)
}
