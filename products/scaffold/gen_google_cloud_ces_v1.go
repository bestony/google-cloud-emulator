package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/ces/v1"
)

func init() {
	registerEntry("google.cloud.ces.v1.SessionService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSessionServiceServer(s, &pb.UnimplementedSessionServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSessionServiceHandlerServer(ctx, mux, &pb.UnimplementedSessionServiceServer{})
		},
	)
	registerEntry("google.cloud.ces.v1.ToolService",
		func(s grpc.ServiceRegistrar) { pb.RegisterToolServiceServer(s, &pb.UnimplementedToolServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterToolServiceHandlerServer(ctx, mux, &pb.UnimplementedToolServiceServer{})
		},
	)
	registerEntry("google.cloud.ces.v1.AgentService",
		func(s grpc.ServiceRegistrar) { pb.RegisterAgentServiceServer(s, &pb.UnimplementedAgentServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAgentServiceHandlerServer(ctx, mux, &pb.UnimplementedAgentServiceServer{})
		},
	)
	registerEntry("google.cloud.ces.v1.WidgetService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterWidgetServiceServer(s, &pb.UnimplementedWidgetServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterWidgetServiceHandlerServer(ctx, mux, &pb.UnimplementedWidgetServiceServer{})
		},
	)
}
