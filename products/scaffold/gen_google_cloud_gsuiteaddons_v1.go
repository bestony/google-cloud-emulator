package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/gsuiteaddons/v1"
)

func init() {
	registerEntry("google.cloud.gsuiteaddons.v1.GSuiteAddOns",
		func(s grpc.ServiceRegistrar) { pb.RegisterGSuiteAddOnsServer(s, &pb.UnimplementedGSuiteAddOnsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterGSuiteAddOnsHandlerServer(ctx, mux, &pb.UnimplementedGSuiteAddOnsServer{})
		},
	)
}
