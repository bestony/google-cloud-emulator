package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/redis/v1"
)

func init() {
	registerEntry("google.cloud.redis.v1.CloudRedis",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudRedisServer(s, &pb.UnimplementedCloudRedisServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCloudRedisHandlerServer(ctx, mux, &pb.UnimplementedCloudRedisServer{})
		},
	)
}
