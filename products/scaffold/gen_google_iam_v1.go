package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/iam/v1"
)

func init() {
	registerEntry("google.iam.v1.IAMPolicy",
		func(s grpc.ServiceRegistrar) { pb.RegisterIAMPolicyServer(s, &pb.UnimplementedIAMPolicyServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterIAMPolicyHandlerServer(ctx, mux, &pb.UnimplementedIAMPolicyServer{})
		},
	)
}
