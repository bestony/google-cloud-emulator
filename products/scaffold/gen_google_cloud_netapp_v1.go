package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/netapp/v1"
)

func init() {
	registerEntry("google.cloud.netapp.v1.NetApp",
		func(s grpc.ServiceRegistrar) { pb.RegisterNetAppServer(s, &pb.UnimplementedNetAppServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNetAppHandlerServer(ctx, mux, &pb.UnimplementedNetAppServer{}) },
	)
}
