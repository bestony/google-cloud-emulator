package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/devicestreaming/v1"
)

func init() {
	registerEntry("google.cloud.devicestreaming.v1.DirectAccessService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDirectAccessServiceServer(s, &pb.UnimplementedDirectAccessServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDirectAccessServiceHandlerServer(ctx, mux, &pb.UnimplementedDirectAccessServiceServer{})
		},
	)
}
