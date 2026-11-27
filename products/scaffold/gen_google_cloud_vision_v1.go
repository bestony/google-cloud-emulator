package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/vision/v1"
)

func init() {
	registerEntry("google.cloud.vision.v1.ProductSearch",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterProductSearchServer(s, &pb.UnimplementedProductSearchServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterProductSearchHandlerServer(ctx, mux, &pb.UnimplementedProductSearchServer{})
		},
	)
	registerEntry("google.cloud.vision.v1.ImageAnnotator",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterImageAnnotatorServer(s, &pb.UnimplementedImageAnnotatorServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterImageAnnotatorHandlerServer(ctx, mux, &pb.UnimplementedImageAnnotatorServer{})
		},
	)
}
