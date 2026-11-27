package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/geminidataanalytics/v1"
)

func init() {
	registerEntry("google.cloud.geminidataanalytics.v1.DataChatService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataChatServiceServer(s, &pb.UnimplementedDataChatServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDataChatServiceHandlerServer(ctx, mux, &pb.UnimplementedDataChatServiceServer{}) },
	)
	registerEntry("google.cloud.geminidataanalytics.v1.DataAgentService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataAgentServiceServer(s, &pb.UnimplementedDataAgentServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDataAgentServiceHandlerServer(ctx, mux, &pb.UnimplementedDataAgentServiceServer{}) },
	)
}
