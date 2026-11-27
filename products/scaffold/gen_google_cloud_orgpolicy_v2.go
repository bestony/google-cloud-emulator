package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/orgpolicy/v2"
)

func init() {
	registerEntry("google.cloud.orgpolicy.v2.OrgPolicy",
		func(s grpc.ServiceRegistrar) { pb.RegisterOrgPolicyServer(s, &pb.UnimplementedOrgPolicyServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterOrgPolicyHandlerServer(ctx, mux, &pb.UnimplementedOrgPolicyServer{})
		},
	)
}
