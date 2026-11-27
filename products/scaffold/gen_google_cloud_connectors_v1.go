package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/connectors/v1"
)

func init() {
	registerEntry("google.cloud.connectors.v1.Connectors",
		func(s grpc.ServiceRegistrar) { pb.RegisterConnectorsServer(s, &pb.UnimplementedConnectorsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterConnectorsHandlerServer(ctx, mux, &pb.UnimplementedConnectorsServer{})
		},
	)
}
