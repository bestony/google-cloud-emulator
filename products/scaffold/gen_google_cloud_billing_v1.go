package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/billing/v1"
)

func init() {
	registerEntry("google.cloud.billing.v1.CloudCatalog",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudCatalogServer(s, &pb.UnimplementedCloudCatalogServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudCatalogHandlerServer(ctx, mux, &pb.UnimplementedCloudCatalogServer{}) },
	)
	registerEntry("google.cloud.billing.v1.CloudBilling",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudBillingServer(s, &pb.UnimplementedCloudBillingServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudBillingHandlerServer(ctx, mux, &pb.UnimplementedCloudBillingServer{}) },
	)
}
