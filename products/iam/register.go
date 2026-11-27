package iam

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	iampb "github.com/bestony/google-cloud-emulator/gen/google/iam/v1"
)

// Register wires the IAMPolicy service into the gRPC server and REST
// gateway.
func Register(grpcServer *grpc.Server, gateway *runtime.ServeMux, state *State) error {
	s := &server{state: state}
	iampb.RegisterIAMPolicyServer(grpcServer, s)
	return iampb.RegisterIAMPolicyHandlerServer(context.Background(), gateway, s)
}
