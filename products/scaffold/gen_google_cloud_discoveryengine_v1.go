package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/discoveryengine/v1"
)

func init() {
	registerEntry("google.cloud.discoveryengine.v1.SearchTuningService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSearchTuningServiceServer(s, &pb.UnimplementedSearchTuningServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSearchTuningServiceHandlerServer(ctx, mux, &pb.UnimplementedSearchTuningServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.UserStoreService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterUserStoreServiceServer(s, &pb.UnimplementedUserStoreServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterUserStoreServiceHandlerServer(ctx, mux, &pb.UnimplementedUserStoreServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.CollectionService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCollectionServiceServer(s, &pb.UnimplementedCollectionServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCollectionServiceHandlerServer(ctx, mux, &pb.UnimplementedCollectionServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.AclConfigService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterAclConfigServiceServer(s, &pb.UnimplementedAclConfigServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAclConfigServiceHandlerServer(ctx, mux, &pb.UnimplementedAclConfigServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.ServingConfigService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterServingConfigServiceServer(s, &pb.UnimplementedServingConfigServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterServingConfigServiceHandlerServer(ctx, mux, &pb.UnimplementedServingConfigServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.RecommendationService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterRecommendationServiceServer(s, &pb.UnimplementedRecommendationServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterRecommendationServiceHandlerServer(ctx, mux, &pb.UnimplementedRecommendationServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.SchemaService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSchemaServiceServer(s, &pb.UnimplementedSchemaServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSchemaServiceHandlerServer(ctx, mux, &pb.UnimplementedSchemaServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.ConversationalSearchService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterConversationalSearchServiceServer(s, &pb.UnimplementedConversationalSearchServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterConversationalSearchServiceHandlerServer(ctx, mux, &pb.UnimplementedConversationalSearchServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.SiteSearchEngineService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSiteSearchEngineServiceServer(s, &pb.UnimplementedSiteSearchEngineServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSiteSearchEngineServiceHandlerServer(ctx, mux, &pb.UnimplementedSiteSearchEngineServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.EngineService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterEngineServiceServer(s, &pb.UnimplementedEngineServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterEngineServiceHandlerServer(ctx, mux, &pb.UnimplementedEngineServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.SessionService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSessionServiceServer(s, &pb.UnimplementedSessionServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSessionServiceHandlerServer(ctx, mux, &pb.UnimplementedSessionServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.AssistantService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterAssistantServiceServer(s, &pb.UnimplementedAssistantServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAssistantServiceHandlerServer(ctx, mux, &pb.UnimplementedAssistantServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.UserEventService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterUserEventServiceServer(s, &pb.UnimplementedUserEventServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterUserEventServiceHandlerServer(ctx, mux, &pb.UnimplementedUserEventServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.IdentityMappingStoreService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterIdentityMappingStoreServiceServer(s, &pb.UnimplementedIdentityMappingStoreServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterIdentityMappingStoreServiceHandlerServer(ctx, mux, &pb.UnimplementedIdentityMappingStoreServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.UserLicenseService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterUserLicenseServiceServer(s, &pb.UnimplementedUserLicenseServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterUserLicenseServiceHandlerServer(ctx, mux, &pb.UnimplementedUserLicenseServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.LicenseConfigService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterLicenseConfigServiceServer(s, &pb.UnimplementedLicenseConfigServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterLicenseConfigServiceHandlerServer(ctx, mux, &pb.UnimplementedLicenseConfigServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.CompletionService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCompletionServiceServer(s, &pb.UnimplementedCompletionServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCompletionServiceHandlerServer(ctx, mux, &pb.UnimplementedCompletionServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.CmekConfigService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterCmekConfigServiceServer(s, &pb.UnimplementedCmekConfigServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterCmekConfigServiceHandlerServer(ctx, mux, &pb.UnimplementedCmekConfigServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.DataConnectorService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataConnectorServiceServer(s, &pb.UnimplementedDataConnectorServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataConnectorServiceHandlerServer(ctx, mux, &pb.UnimplementedDataConnectorServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.DocumentService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDocumentServiceServer(s, &pb.UnimplementedDocumentServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDocumentServiceHandlerServer(ctx, mux, &pb.UnimplementedDocumentServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.DataStoreService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataStoreServiceServer(s, &pb.UnimplementedDataStoreServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataStoreServiceHandlerServer(ctx, mux, &pb.UnimplementedDataStoreServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.SearchService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSearchServiceServer(s, &pb.UnimplementedSearchServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSearchServiceHandlerServer(ctx, mux, &pb.UnimplementedSearchServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.ProjectService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterProjectServiceServer(s, &pb.UnimplementedProjectServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterProjectServiceHandlerServer(ctx, mux, &pb.UnimplementedProjectServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.ControlService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterControlServiceServer(s, &pb.UnimplementedControlServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterControlServiceHandlerServer(ctx, mux, &pb.UnimplementedControlServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.GroundedGenerationService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterGroundedGenerationServiceServer(s, &pb.UnimplementedGroundedGenerationServiceServer{})
		},
		nil,
	)
	registerEntry("google.cloud.discoveryengine.v1.RankService",
		func(s grpc.ServiceRegistrar) { pb.RegisterRankServiceServer(s, &pb.UnimplementedRankServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterRankServiceHandlerServer(ctx, mux, &pb.UnimplementedRankServiceServer{})
		},
	)
	registerEntry("google.cloud.discoveryengine.v1.WidgetConfigService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterWidgetConfigServiceServer(s, &pb.UnimplementedWidgetConfigServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterWidgetConfigServiceHandlerServer(ctx, mux, &pb.UnimplementedWidgetConfigServiceServer{})
		},
	)
}
