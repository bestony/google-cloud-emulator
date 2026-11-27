package gke

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	containerpb "github.com/bestony/google-cloud-emulator/gen/google/container/v1"
)

// Register wires the implemented ClusterManager methods into the gRPC
// server and REST gateway. All other container.v1 methods remain
// scaffolded UNIMPLEMENTED via the embedded generated server.
func Register(grpcServer *grpc.Server, gateway *runtime.ServeMux, state *State) error {
	s := &server{state: state}
	containerpb.RegisterClusterManagerServer(grpcServer, s)
	return containerpb.RegisterClusterManagerHandlerServer(context.Background(), gateway, s)
}
