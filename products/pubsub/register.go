package pubsub

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	pubsubpb "github.com/bestony/google-cloud-emulator/gen/google/pubsub/v1"
)

// Register wires the Pub/Sub gRPC services and their REST gateway handlers
// into the shared servers.
func Register(grpcServer *grpc.Server, gateway *runtime.ServeMux, st *State) error {
	pub := NewPublisherServer(st)
	sub := NewSubscriberServer(st)
	pubsubpb.RegisterPublisherServer(grpcServer, pub)
	pubsubpb.RegisterSubscriberServer(grpcServer, sub)

	ctx := context.Background()
	if err := pubsubpb.RegisterPublisherHandlerServer(ctx, gateway, pub); err != nil {
		return err
	}
	return pubsubpb.RegisterSubscriberHandlerServer(ctx, gateway, sub)
}
