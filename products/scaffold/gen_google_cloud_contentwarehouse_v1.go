package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/contentwarehouse/v1"
)

func init() {
	registerEntry("google.cloud.contentwarehouse.v1.PipelineService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterPipelineServiceServer(s, &pb.UnimplementedPipelineServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterPipelineServiceHandlerServer(ctx, mux, &pb.UnimplementedPipelineServiceServer{})
		},
	)
	registerEntry("google.cloud.contentwarehouse.v1.SynonymSetService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSynonymSetServiceServer(s, &pb.UnimplementedSynonymSetServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSynonymSetServiceHandlerServer(ctx, mux, &pb.UnimplementedSynonymSetServiceServer{})
		},
	)
	registerEntry("google.cloud.contentwarehouse.v1.DocumentSchemaService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDocumentSchemaServiceServer(s, &pb.UnimplementedDocumentSchemaServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDocumentSchemaServiceHandlerServer(ctx, mux, &pb.UnimplementedDocumentSchemaServiceServer{})
		},
	)
	registerEntry("google.cloud.contentwarehouse.v1.DocumentLinkService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDocumentLinkServiceServer(s, &pb.UnimplementedDocumentLinkServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDocumentLinkServiceHandlerServer(ctx, mux, &pb.UnimplementedDocumentLinkServiceServer{})
		},
	)
	registerEntry("google.cloud.contentwarehouse.v1.RuleSetService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterRuleSetServiceServer(s, &pb.UnimplementedRuleSetServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterRuleSetServiceHandlerServer(ctx, mux, &pb.UnimplementedRuleSetServiceServer{})
		},
	)
	registerEntry("google.cloud.contentwarehouse.v1.DocumentService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDocumentServiceServer(s, &pb.UnimplementedDocumentServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDocumentServiceHandlerServer(ctx, mux, &pb.UnimplementedDocumentServiceServer{})
		},
	)
}
