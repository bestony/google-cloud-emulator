package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/visionai/v1"
)

func init() {
	registerEntry("google.cloud.visionai.v1.LiveVideoAnalytics",
		func(s grpc.ServiceRegistrar) { pb.RegisterLiveVideoAnalyticsServer(s, &pb.UnimplementedLiveVideoAnalyticsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterLiveVideoAnalyticsHandlerServer(ctx, mux, &pb.UnimplementedLiveVideoAnalyticsServer{}) },
	)
	registerEntry("google.cloud.visionai.v1.StreamsService",
		func(s grpc.ServiceRegistrar) { pb.RegisterStreamsServiceServer(s, &pb.UnimplementedStreamsServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterStreamsServiceHandlerServer(ctx, mux, &pb.UnimplementedStreamsServiceServer{}) },
	)
	registerEntry("google.cloud.visionai.v1.Warehouse",
		func(s grpc.ServiceRegistrar) { pb.RegisterWarehouseServer(s, &pb.UnimplementedWarehouseServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterWarehouseHandlerServer(ctx, mux, &pb.UnimplementedWarehouseServer{}) },
	)
	registerEntry("google.cloud.visionai.v1.StreamingService",
		func(s grpc.ServiceRegistrar) { pb.RegisterStreamingServiceServer(s, &pb.UnimplementedStreamingServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterStreamingServiceHandlerServer(ctx, mux, &pb.UnimplementedStreamingServiceServer{}) },
	)
	registerEntry("google.cloud.visionai.v1.AppPlatform",
		func(s grpc.ServiceRegistrar) { pb.RegisterAppPlatformServer(s, &pb.UnimplementedAppPlatformServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAppPlatformHandlerServer(ctx, mux, &pb.UnimplementedAppPlatformServer{}) },
	)
	registerEntry("google.cloud.visionai.v1.HealthCheckService",
		func(s grpc.ServiceRegistrar) { pb.RegisterHealthCheckServiceServer(s, &pb.UnimplementedHealthCheckServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterHealthCheckServiceHandlerServer(ctx, mux, &pb.UnimplementedHealthCheckServiceServer{}) },
	)
}
