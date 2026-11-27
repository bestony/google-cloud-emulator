package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/metastore/v1"
)

func init() {
	registerEntry("google.cloud.metastore.v1.DataprocMetastore",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataprocMetastoreServer(s, &pb.UnimplementedDataprocMetastoreServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDataprocMetastoreHandlerServer(ctx, mux, &pb.UnimplementedDataprocMetastoreServer{}) },
	)
	registerEntry("google.cloud.metastore.v1.DataprocMetastoreFederation",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataprocMetastoreFederationServer(s, &pb.UnimplementedDataprocMetastoreFederationServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDataprocMetastoreFederationHandlerServer(ctx, mux, &pb.UnimplementedDataprocMetastoreFederationServer{}) },
	)
}
