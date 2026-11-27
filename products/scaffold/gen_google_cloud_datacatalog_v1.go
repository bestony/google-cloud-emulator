package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/datacatalog/v1"
)

func init() {
	registerEntry("google.cloud.datacatalog.v1.PolicyTagManager",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterPolicyTagManagerServer(s, &pb.UnimplementedPolicyTagManagerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterPolicyTagManagerHandlerServer(ctx, mux, &pb.UnimplementedPolicyTagManagerServer{})
		},
	)
	registerEntry("google.cloud.datacatalog.v1.DataCatalog",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataCatalogServer(s, &pb.UnimplementedDataCatalogServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataCatalogHandlerServer(ctx, mux, &pb.UnimplementedDataCatalogServer{})
		},
	)
	registerEntry("google.cloud.datacatalog.v1.PolicyTagManagerSerialization",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterPolicyTagManagerSerializationServer(s, &pb.UnimplementedPolicyTagManagerSerializationServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterPolicyTagManagerSerializationHandlerServer(ctx, mux, &pb.UnimplementedPolicyTagManagerSerializationServer{})
		},
	)
}
