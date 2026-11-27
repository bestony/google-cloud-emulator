package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/texttospeech/v1"
)

func init() {
	registerEntry("google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize",
		func(s grpc.ServiceRegistrar) { pb.RegisterTextToSpeechLongAudioSynthesizeServer(s, &pb.UnimplementedTextToSpeechLongAudioSynthesizeServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTextToSpeechLongAudioSynthesizeHandlerServer(ctx, mux, &pb.UnimplementedTextToSpeechLongAudioSynthesizeServer{}) },
	)
	registerEntry("google.cloud.texttospeech.v1.TextToSpeech",
		func(s grpc.ServiceRegistrar) { pb.RegisterTextToSpeechServer(s, &pb.UnimplementedTextToSpeechServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTextToSpeechHandlerServer(ctx, mux, &pb.UnimplementedTextToSpeechServer{}) },
	)
}
