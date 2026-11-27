package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/pubsublite/v1"
)

func init() {
	registerEntry("google.cloud.pubsublite.v1.PublisherService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterPublisherServiceServer(s, &pb.UnimplementedPublisherServiceServer{})
		},
		nil,
	)
	registerEntry("google.cloud.pubsublite.v1.TopicStatsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterTopicStatsServiceServer(s, &pb.UnimplementedTopicStatsServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterTopicStatsServiceHandlerServer(ctx, mux, &pb.UnimplementedTopicStatsServiceServer{})
		},
	)
	registerEntry("google.cloud.pubsublite.v1.CursorService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCursorServiceServer(s, &pb.UnimplementedCursorServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCursorServiceHandlerServer(ctx, mux, &pb.UnimplementedCursorServiceServer{})
		},
	)
	registerEntry("google.cloud.pubsublite.v1.AdminService",
		func(s grpc.ServiceRegistrar) { pb.RegisterAdminServiceServer(s, &pb.UnimplementedAdminServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAdminServiceHandlerServer(ctx, mux, &pb.UnimplementedAdminServiceServer{})
		},
	)
	registerEntry("google.cloud.pubsublite.v1.SubscriberService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSubscriberServiceServer(s, &pb.UnimplementedSubscriberServiceServer{})
		},
		nil,
	)
	registerEntry("google.cloud.pubsublite.v1.PartitionAssignmentService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterPartitionAssignmentServiceServer(s, &pb.UnimplementedPartitionAssignmentServiceServer{})
		},
		nil,
	)
}
