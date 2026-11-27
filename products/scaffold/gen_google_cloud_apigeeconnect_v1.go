package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/apigeeconnect/v1"
)

func init() {
	registerEntry("google.cloud.apigeeconnect.v1.ConnectionService",
		func(s grpc.ServiceRegistrar) { pb.RegisterConnectionServiceServer(s, &pb.UnimplementedConnectionServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterConnectionServiceHandlerServer(ctx, mux, &pb.UnimplementedConnectionServiceServer{}) },
	)
	registerEntry("google.cloud.apigeeconnect.v1.Tether",
		func(s grpc.ServiceRegistrar) { pb.RegisterTetherServer(s, &pb.UnimplementedTetherServer{}) },
		nil,
	)
}
