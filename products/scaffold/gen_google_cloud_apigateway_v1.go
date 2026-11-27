package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/apigateway/v1"
)

func init() {
	registerEntry("google.cloud.apigateway.v1.ApiGatewayService",
		func(s grpc.ServiceRegistrar) { pb.RegisterApiGatewayServiceServer(s, &pb.UnimplementedApiGatewayServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterApiGatewayServiceHandlerServer(ctx, mux, &pb.UnimplementedApiGatewayServiceServer{}) },
	)
}
