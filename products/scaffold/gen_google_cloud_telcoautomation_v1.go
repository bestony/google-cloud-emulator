package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/telcoautomation/v1"
)

func init() {
	registerEntry("google.cloud.telcoautomation.v1.TelcoAutomation",
		func(s grpc.ServiceRegistrar) { pb.RegisterTelcoAutomationServer(s, &pb.UnimplementedTelcoAutomationServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTelcoAutomationHandlerServer(ctx, mux, &pb.UnimplementedTelcoAutomationServer{}) },
	)
}
