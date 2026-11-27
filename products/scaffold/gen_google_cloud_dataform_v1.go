package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/dataform/v1"
)

func init() {
	registerEntry("google.cloud.dataform.v1.Dataform",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataformServer(s, &pb.UnimplementedDataformServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataformHandlerServer(ctx, mux, &pb.UnimplementedDataformServer{})
		},
	)
}
