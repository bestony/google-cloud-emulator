package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/accessapproval/v1"
)

func init() {
	registerEntry("google.cloud.accessapproval.v1.AccessApproval",
		func(s grpc.ServiceRegistrar) { pb.RegisterAccessApprovalServer(s, &pb.UnimplementedAccessApprovalServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAccessApprovalHandlerServer(ctx, mux, &pb.UnimplementedAccessApprovalServer{}) },
	)
}
