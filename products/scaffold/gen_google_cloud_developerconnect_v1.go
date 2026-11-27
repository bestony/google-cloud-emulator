package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/developerconnect/v1"
)

func init() {
	registerEntry("google.cloud.developerconnect.v1.DeveloperConnect",
		func(s grpc.ServiceRegistrar) { pb.RegisterDeveloperConnectServer(s, &pb.UnimplementedDeveloperConnectServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDeveloperConnectHandlerServer(ctx, mux, &pb.UnimplementedDeveloperConnectServer{}) },
	)
	registerEntry("google.cloud.developerconnect.v1.InsightsConfigService",
		func(s grpc.ServiceRegistrar) { pb.RegisterInsightsConfigServiceServer(s, &pb.UnimplementedInsightsConfigServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInsightsConfigServiceHandlerServer(ctx, mux, &pb.UnimplementedInsightsConfigServiceServer{}) },
	)
}
