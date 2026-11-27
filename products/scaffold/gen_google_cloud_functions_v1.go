package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/functions/v1"
)

func init() {
	registerEntry("google.cloud.functions.v1.CloudFunctionsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCloudFunctionsServiceServer(s, &pb.UnimplementedCloudFunctionsServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCloudFunctionsServiceHandlerServer(ctx, mux, &pb.UnimplementedCloudFunctionsServiceServer{})
		},
	)
}
