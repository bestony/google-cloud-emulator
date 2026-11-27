package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/osconfig/v1"
)

func init() {
	registerEntry("google.cloud.osconfig.v1.OsConfigService",
		func(s grpc.ServiceRegistrar) { pb.RegisterOsConfigServiceServer(s, &pb.UnimplementedOsConfigServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterOsConfigServiceHandlerServer(ctx, mux, &pb.UnimplementedOsConfigServiceServer{}) },
	)
	registerEntry("google.cloud.osconfig.v1.OsConfigZonalService",
		func(s grpc.ServiceRegistrar) { pb.RegisterOsConfigZonalServiceServer(s, &pb.UnimplementedOsConfigZonalServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterOsConfigZonalServiceHandlerServer(ctx, mux, &pb.UnimplementedOsConfigZonalServiceServer{}) },
	)
}
