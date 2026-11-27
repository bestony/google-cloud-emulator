package datastore

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	dspb "github.com/bestony/google-cloud-emulator/gen/google/datastore/v1"
)

// Register wires the Datastore gRPC service and generated REST gateway.
func Register(grpcServer *grpc.Server, gateway *runtime.ServeMux, st *State) error {
	srv := NewServer(st)
	dspb.RegisterDatastoreServer(grpcServer, srv)
	return dspb.RegisterDatastoreHandlerServer(context.Background(), gateway, srv)
}
