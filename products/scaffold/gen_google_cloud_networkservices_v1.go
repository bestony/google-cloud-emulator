package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/networkservices/v1"
)

func init() {
	registerEntry("google.cloud.networkservices.v1.NetworkServices",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterNetworkServicesServer(s, &pb.UnimplementedNetworkServicesServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterNetworkServicesHandlerServer(ctx, mux, &pb.UnimplementedNetworkServicesServer{})
		},
	)
	registerEntry("google.cloud.networkservices.v1.DepService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDepServiceServer(s, &pb.UnimplementedDepServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDepServiceHandlerServer(ctx, mux, &pb.UnimplementedDepServiceServer{})
		},
	)
}
