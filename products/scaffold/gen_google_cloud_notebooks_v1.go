package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/notebooks/v1"
)

func init() {
	registerEntry("google.cloud.notebooks.v1.NotebookService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterNotebookServiceServer(s, &pb.UnimplementedNotebookServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterNotebookServiceHandlerServer(ctx, mux, &pb.UnimplementedNotebookServiceServer{})
		},
	)
	registerEntry("google.cloud.notebooks.v1.ManagedNotebookService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterManagedNotebookServiceServer(s, &pb.UnimplementedManagedNotebookServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterManagedNotebookServiceHandlerServer(ctx, mux, &pb.UnimplementedManagedNotebookServiceServer{})
		},
	)
}
