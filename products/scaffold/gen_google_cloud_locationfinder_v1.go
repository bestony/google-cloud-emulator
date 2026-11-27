package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/locationfinder/v1"
)

func init() {
	registerEntry("google.cloud.locationfinder.v1.CloudLocationFinder",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCloudLocationFinderServer(s, &pb.UnimplementedCloudLocationFinderServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCloudLocationFinderHandlerServer(ctx, mux, &pb.UnimplementedCloudLocationFinderServer{})
		},
	)
}
