package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/language/v1"
)

func init() {
	registerEntry("google.cloud.language.v1.LanguageService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterLanguageServiceServer(s, &pb.UnimplementedLanguageServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterLanguageServiceHandlerServer(ctx, mux, &pb.UnimplementedLanguageServiceServer{})
		},
	)
}
