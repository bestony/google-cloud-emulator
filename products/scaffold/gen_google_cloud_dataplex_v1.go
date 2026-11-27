package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/dataplex/v1"
)

func init() {
	registerEntry("google.cloud.dataplex.v1.DataplexService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataplexServiceServer(s, &pb.UnimplementedDataplexServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataplexServiceHandlerServer(ctx, mux, &pb.UnimplementedDataplexServiceServer{})
		},
	)
	registerEntry("google.cloud.dataplex.v1.MetadataService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterMetadataServiceServer(s, &pb.UnimplementedMetadataServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterMetadataServiceHandlerServer(ctx, mux, &pb.UnimplementedMetadataServiceServer{})
		},
	)
	registerEntry("google.cloud.dataplex.v1.BusinessGlossaryService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterBusinessGlossaryServiceServer(s, &pb.UnimplementedBusinessGlossaryServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterBusinessGlossaryServiceHandlerServer(ctx, mux, &pb.UnimplementedBusinessGlossaryServiceServer{})
		},
	)
	registerEntry("google.cloud.dataplex.v1.DataTaxonomyService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataTaxonomyServiceServer(s, &pb.UnimplementedDataTaxonomyServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataTaxonomyServiceHandlerServer(ctx, mux, &pb.UnimplementedDataTaxonomyServiceServer{})
		},
	)
	registerEntry("google.cloud.dataplex.v1.ContentService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterContentServiceServer(s, &pb.UnimplementedContentServiceServer{})
		},
		nil,
	)
	registerEntry("google.cloud.dataplex.v1.DataScanService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataScanServiceServer(s, &pb.UnimplementedDataScanServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataScanServiceHandlerServer(ctx, mux, &pb.UnimplementedDataScanServiceServer{})
		},
	)
	registerEntry("google.cloud.dataplex.v1.CmekService",
		func(s grpc.ServiceRegistrar) { pb.RegisterCmekServiceServer(s, &pb.UnimplementedCmekServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCmekServiceHandlerServer(ctx, mux, &pb.UnimplementedCmekServiceServer{})
		},
	)
	registerEntry("google.cloud.dataplex.v1.CatalogService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCatalogServiceServer(s, &pb.UnimplementedCatalogServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCatalogServiceHandlerServer(ctx, mux, &pb.UnimplementedCatalogServiceServer{})
		},
	)
	registerEntry("google.cloud.dataplex.v1.DataProductService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataProductServiceServer(s, &pb.UnimplementedDataProductServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataProductServiceHandlerServer(ctx, mux, &pb.UnimplementedDataProductServiceServer{})
		},
	)
}
