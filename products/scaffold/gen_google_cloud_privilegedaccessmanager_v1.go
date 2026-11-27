package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/privilegedaccessmanager/v1"
)

func init() {
	registerEntry("google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager",
		func(s grpc.ServiceRegistrar) { pb.RegisterPrivilegedAccessManagerServer(s, &pb.UnimplementedPrivilegedAccessManagerServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterPrivilegedAccessManagerHandlerServer(ctx, mux, &pb.UnimplementedPrivilegedAccessManagerServer{}) },
	)
}
