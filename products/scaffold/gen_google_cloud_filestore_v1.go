package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/filestore/v1"
)

func init() {
	registerEntry("google.cloud.filestore.v1.CloudFilestoreManager",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCloudFilestoreManagerServer(s, &pb.UnimplementedCloudFilestoreManagerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCloudFilestoreManagerHandlerServer(ctx, mux, &pb.UnimplementedCloudFilestoreManagerServer{})
		},
	)
}
