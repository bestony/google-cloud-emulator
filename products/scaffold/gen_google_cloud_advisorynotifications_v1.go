package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/advisorynotifications/v1"
)

func init() {
	registerEntry("google.cloud.advisorynotifications.v1.AdvisoryNotificationsService",
		func(s grpc.ServiceRegistrar) { pb.RegisterAdvisoryNotificationsServiceServer(s, &pb.UnimplementedAdvisoryNotificationsServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAdvisoryNotificationsServiceHandlerServer(ctx, mux, &pb.UnimplementedAdvisoryNotificationsServiceServer{}) },
	)
}
