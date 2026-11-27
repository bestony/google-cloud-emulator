package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/policytroubleshooter/v1"
)

func init() {
	registerEntry("google.cloud.policytroubleshooter.v1.IamChecker",
		func(s grpc.ServiceRegistrar) { pb.RegisterIamCheckerServer(s, &pb.UnimplementedIamCheckerServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterIamCheckerHandlerServer(ctx, mux, &pb.UnimplementedIamCheckerServer{})
		},
	)
}
