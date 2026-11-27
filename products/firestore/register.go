package firestore

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	fspb "github.com/bestony/google-cloud-emulator/gen/google/firestore/v1"
)

// Register wires the Firestore gRPC service. Firestore's generated protos do
// not include grpc-gateway bindings in this repository, so REST is served by
// the manual HTTP handler mounted by cmd/gcpex.
func Register(grpcServer *grpc.Server, _ *runtime.ServeMux, st *State) error {
	fspb.RegisterFirestoreServer(grpcServer, NewServer(st))
	return nil
}
