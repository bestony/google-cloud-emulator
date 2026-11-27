package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/speech/v1"
)

func init() {
	registerEntry("google.cloud.speech.v1.Speech",
		func(s grpc.ServiceRegistrar) { pb.RegisterSpeechServer(s, &pb.UnimplementedSpeechServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSpeechHandlerServer(ctx, mux, &pb.UnimplementedSpeechServer{}) },
	)
	registerEntry("google.cloud.speech.v1.Adaptation",
		func(s grpc.ServiceRegistrar) { pb.RegisterAdaptationServer(s, &pb.UnimplementedAdaptationServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAdaptationHandlerServer(ctx, mux, &pb.UnimplementedAdaptationServer{}) },
	)
}
