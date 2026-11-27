// Package scaffold registers every generated gRPC service that has no
// hand-written implementation. All methods return codes.Unimplemented (501
// over REST), keeping unknown surfaces honest while making every product
// reachable.
package scaffold

import (
	"context"
	"errors"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type entry struct {
	name string
	grpc func(grpc.ServiceRegistrar)
	gw   func(context.Context, *runtime.ServeMux) error
}

var entries []entry
var seen = make(map[string]bool)

func registerEntry(name string, grpcReg func(grpc.ServiceRegistrar), gwReg func(context.Context, *runtime.ServeMux) error) {
	if name != "" {
		if seen[name] {
			return
		}
		seen[name] = true
	}
	entries = append(entries, entry{name: name, grpc: grpcReg, gw: gwReg})
}

// RegisterAll registers all scaffolded services. Gateway registration errors
// (e.g. route conflicts) are skipped so gRPC keeps working; they are returned
// joined for logging.
func RegisterAll(grpcServer *grpc.Server, gateway *runtime.ServeMux) error {
	var errs []error
	for _, e := range entries {
		if e.grpc != nil {
			e.grpc(grpcServer)
		}
		if e.gw != nil {
			if err := e.gw(context.Background(), gateway); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
