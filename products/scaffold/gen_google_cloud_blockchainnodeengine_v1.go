package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/blockchainnodeengine/v1"
)

func init() {
	registerEntry("google.cloud.blockchainnodeengine.v1.BlockchainNodeEngine",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterBlockchainNodeEngineServer(s, &pb.UnimplementedBlockchainNodeEngineServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterBlockchainNodeEngineHandlerServer(ctx, mux, &pb.UnimplementedBlockchainNodeEngineServer{})
		},
	)
}
