package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/storagetransfer/v1"
)

func init() {
	registerEntry("google.storagetransfer.v1.StorageTransferService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterStorageTransferServiceServer(s, &pb.UnimplementedStorageTransferServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterStorageTransferServiceHandlerServer(ctx, mux, &pb.UnimplementedStorageTransferServiceServer{})
		},
	)
}
