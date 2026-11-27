package secretmanager

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	smpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/secretmanager/v1"
)

// Register wires the Secret Manager gRPC service and its REST gateway
// handlers into the shared servers.
func Register(grpcServer *grpc.Server, gateway *runtime.ServeMux, st *State) error {
	srv := NewServer(st)
	ops := NewOperationsServer()
	smpb.RegisterSecretManagerServiceServer(grpcServer, srv)
	smpb.RegisterSecretOperationsServiceServer(grpcServer, ops)

	ctx := context.Background()
	if err := smpb.RegisterSecretManagerServiceHandlerServer(ctx, gateway, srv); err != nil {
		return err
	}
	return smpb.RegisterSecretOperationsServiceHandlerServer(ctx, gateway, ops)
}
