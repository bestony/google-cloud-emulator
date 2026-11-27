package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/universalledger/v1"
)

func init() {
	registerEntry("google.cloud.universalledger.v1.UniversalLedger",
		func(s grpc.ServiceRegistrar) { pb.RegisterUniversalLedgerServer(s, &pb.UnimplementedUniversalLedgerServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterUniversalLedgerHandlerServer(ctx, mux, &pb.UnimplementedUniversalLedgerServer{}) },
	)
}
