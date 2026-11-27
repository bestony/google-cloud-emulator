package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/contactcenterinsights/v1"
)

func init() {
	registerEntry("google.cloud.contactcenterinsights.v1.ContactCenterInsights",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterContactCenterInsightsServer(s, &pb.UnimplementedContactCenterInsightsServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterContactCenterInsightsHandlerServer(ctx, mux, &pb.UnimplementedContactCenterInsightsServer{})
		},
	)
}
