package run

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	runpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/run/v2"
)

// Register wires the Cloud Run v2 Services gRPC service and its REST gateway
// handlers into the shared servers.
func Register(grpcServer *grpc.Server, gateway *runtime.ServeMux, st *State) error {
	srv := NewServer(st)
	runpb.RegisterServicesServer(grpcServer, srv)
	return runpb.RegisterServicesHandlerServer(context.Background(), gateway, srv)
}
