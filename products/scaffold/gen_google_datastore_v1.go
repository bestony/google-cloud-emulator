package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/datastore/v1"
)

func init() {
	registerEntry("google.datastore.v1.Datastore",
		func(s grpc.ServiceRegistrar) { pb.RegisterDatastoreServer(s, &pb.UnimplementedDatastoreServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDatastoreHandlerServer(ctx, mux, &pb.UnimplementedDatastoreServer{})
		},
	)
}
