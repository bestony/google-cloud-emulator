package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/recommender/v1"
)

func init() {
	registerEntry("google.cloud.recommender.v1.Recommender",
		func(s grpc.ServiceRegistrar) { pb.RegisterRecommenderServer(s, &pb.UnimplementedRecommenderServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRecommenderHandlerServer(ctx, mux, &pb.UnimplementedRecommenderServer{}) },
	)
}
