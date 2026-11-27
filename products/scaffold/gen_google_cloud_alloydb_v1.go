package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/alloydb/v1"
)

func init() {
	registerEntry("google.cloud.alloydb.v1.AlloyDBAdmin",
		func(s grpc.ServiceRegistrar) { pb.RegisterAlloyDBAdminServer(s, &pb.UnimplementedAlloyDBAdminServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAlloyDBAdminHandlerServer(ctx, mux, &pb.UnimplementedAlloyDBAdminServer{}) },
	)
	registerEntry("google.cloud.alloydb.v1.AlloyDBCSQLAdmin",
		func(s grpc.ServiceRegistrar) { pb.RegisterAlloyDBCSQLAdminServer(s, &pb.UnimplementedAlloyDBCSQLAdminServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAlloyDBCSQLAdminHandlerServer(ctx, mux, &pb.UnimplementedAlloyDBCSQLAdminServer{}) },
	)
}
