package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/vectorsearch/v1"
)

func init() {
	registerEntry("google.cloud.vectorsearch.v1.DataObjectService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataObjectServiceServer(s, &pb.UnimplementedDataObjectServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDataObjectServiceHandlerServer(ctx, mux, &pb.UnimplementedDataObjectServiceServer{}) },
	)
	registerEntry("google.cloud.vectorsearch.v1.DataObjectSearchService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataObjectSearchServiceServer(s, &pb.UnimplementedDataObjectSearchServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDataObjectSearchServiceHandlerServer(ctx, mux, &pb.UnimplementedDataObjectSearchServiceServer{}) },
	)
	registerEntry("google.cloud.vectorsearch.v1.VectorSearchService",
		func(s grpc.ServiceRegistrar) { pb.RegisterVectorSearchServiceServer(s, &pb.UnimplementedVectorSearchServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterVectorSearchServiceHandlerServer(ctx, mux, &pb.UnimplementedVectorSearchServiceServer{}) },
	)
}
