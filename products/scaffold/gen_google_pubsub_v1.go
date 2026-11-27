package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/pubsub/v1"
)

func init() {
	registerEntry("google.pubsub.v1.SchemaService",
		func(s grpc.ServiceRegistrar) { pb.RegisterSchemaServiceServer(s, &pb.UnimplementedSchemaServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSchemaServiceHandlerServer(ctx, mux, &pb.UnimplementedSchemaServiceServer{}) },
	)
}
