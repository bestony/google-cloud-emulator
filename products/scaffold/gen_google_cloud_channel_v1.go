package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/channel/v1"
)

func init() {
	registerEntry("google.cloud.channel.v1.CloudChannelService",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudChannelServiceServer(s, &pb.UnimplementedCloudChannelServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudChannelServiceHandlerServer(ctx, mux, &pb.UnimplementedCloudChannelServiceServer{}) },
	)
	registerEntry("google.cloud.channel.v1.CloudChannelReportsService",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudChannelReportsServiceServer(s, &pb.UnimplementedCloudChannelReportsServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudChannelReportsServiceHandlerServer(ctx, mux, &pb.UnimplementedCloudChannelReportsServiceServer{}) },
	)
}
