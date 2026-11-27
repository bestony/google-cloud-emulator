package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/iot/v1"
)

func init() {
	registerEntry("google.cloud.iot.v1.DeviceManager",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDeviceManagerServer(s, &pb.UnimplementedDeviceManagerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDeviceManagerHandlerServer(ctx, mux, &pb.UnimplementedDeviceManagerServer{})
		},
	)
}
