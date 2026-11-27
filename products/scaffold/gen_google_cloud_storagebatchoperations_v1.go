package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/storagebatchoperations/v1"
)

func init() {
	registerEntry("google.cloud.storagebatchoperations.v1.StorageBatchOperations",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterStorageBatchOperationsServer(s, &pb.UnimplementedStorageBatchOperationsServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterStorageBatchOperationsHandlerServer(ctx, mux, &pb.UnimplementedStorageBatchOperationsServer{})
		},
	)
}
