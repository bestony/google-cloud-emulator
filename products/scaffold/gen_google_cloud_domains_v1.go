package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/domains/v1"
)

func init() {
	registerEntry("google.cloud.domains.v1.Domains",
		func(s grpc.ServiceRegistrar) { pb.RegisterDomainsServer(s, &pb.UnimplementedDomainsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDomainsHandlerServer(ctx, mux, &pb.UnimplementedDomainsServer{}) },
	)
}
