package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/documentai/v1"
)

func init() {
	registerEntry("google.cloud.documentai.v1.DocumentProcessorService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDocumentProcessorServiceServer(s, &pb.UnimplementedDocumentProcessorServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDocumentProcessorServiceHandlerServer(ctx, mux, &pb.UnimplementedDocumentProcessorServiceServer{}) },
	)
}
