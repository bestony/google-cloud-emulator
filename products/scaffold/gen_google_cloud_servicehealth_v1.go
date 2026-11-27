package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/servicehealth/v1"
)

func init() {
	registerEntry("google.cloud.servicehealth.v1.ServiceHealth",
		func(s grpc.ServiceRegistrar) { pb.RegisterServiceHealthServer(s, &pb.UnimplementedServiceHealthServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterServiceHealthHandlerServer(ctx, mux, &pb.UnimplementedServiceHealthServer{}) },
	)
}
