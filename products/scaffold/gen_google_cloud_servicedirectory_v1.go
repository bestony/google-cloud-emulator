package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/servicedirectory/v1"
)

func init() {
	registerEntry("google.cloud.servicedirectory.v1.LookupService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterLookupServiceServer(s, &pb.UnimplementedLookupServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterLookupServiceHandlerServer(ctx, mux, &pb.UnimplementedLookupServiceServer{})
		},
	)
	registerEntry("google.cloud.servicedirectory.v1.RegistrationService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterRegistrationServiceServer(s, &pb.UnimplementedRegistrationServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterRegistrationServiceHandlerServer(ctx, mux, &pb.UnimplementedRegistrationServiceServer{})
		},
	)
}
