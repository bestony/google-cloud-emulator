package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/chat/v1"
)

func init() {
	registerEntry("google.chat.v1.ChatService",
		func(s grpc.ServiceRegistrar) { pb.RegisterChatServiceServer(s, &pb.UnimplementedChatServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterChatServiceHandlerServer(ctx, mux, &pb.UnimplementedChatServiceServer{})
		},
	)
}
