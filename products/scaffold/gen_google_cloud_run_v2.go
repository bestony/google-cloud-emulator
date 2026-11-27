package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/run/v2"
)

func init() {
	registerEntry("google.cloud.run.v2.Revisions",
		func(s grpc.ServiceRegistrar) { pb.RegisterRevisionsServer(s, &pb.UnimplementedRevisionsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterRevisionsHandlerServer(ctx, mux, &pb.UnimplementedRevisionsServer{})
		},
	)
}
