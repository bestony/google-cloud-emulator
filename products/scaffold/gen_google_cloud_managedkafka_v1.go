package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/managedkafka/v1"
)

func init() {
	registerEntry("google.cloud.managedkafka.v1.ManagedKafkaConnect",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterManagedKafkaConnectServer(s, &pb.UnimplementedManagedKafkaConnectServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterManagedKafkaConnectHandlerServer(ctx, mux, &pb.UnimplementedManagedKafkaConnectServer{})
		},
	)
	registerEntry("google.cloud.managedkafka.v1.ManagedKafka",
		func(s grpc.ServiceRegistrar) { pb.RegisterManagedKafkaServer(s, &pb.UnimplementedManagedKafkaServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterManagedKafkaHandlerServer(ctx, mux, &pb.UnimplementedManagedKafkaServer{})
		},
	)
}
