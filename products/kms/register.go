package kms

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	kmspb "github.com/bestony/google-cloud-emulator/gen/google/cloud/kms/v1"
)

// Register wires the Cloud KMS gRPC service and its REST gateway handlers
// into the shared servers.
func Register(grpcServer *grpc.Server, gateway *runtime.ServeMux, st *State) error {
	srv := NewServer(st)
	kmspb.RegisterKeyManagementServiceServer(grpcServer, srv)
	return kmspb.RegisterKeyManagementServiceHandlerServer(context.Background(), gateway, srv)
}
