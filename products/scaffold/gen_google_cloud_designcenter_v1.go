package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/designcenter/v1"
)

func init() {
	registerEntry("google.cloud.designcenter.v1.DesignCenter",
		func(s grpc.ServiceRegistrar) { pb.RegisterDesignCenterServer(s, &pb.UnimplementedDesignCenterServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDesignCenterHandlerServer(ctx, mux, &pb.UnimplementedDesignCenterServer{})
		},
	)
}
