package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/vpcaccess/v1"
)

func init() {
	registerEntry("google.cloud.vpcaccess.v1.VpcAccessService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterVpcAccessServiceServer(s, &pb.UnimplementedVpcAccessServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterVpcAccessServiceHandlerServer(ctx, mux, &pb.UnimplementedVpcAccessServiceServer{})
		},
	)
}
