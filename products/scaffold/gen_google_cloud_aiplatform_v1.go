package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/aiplatform/v1"
)

func init() {
	registerEntry("google.cloud.aiplatform.v1.GenAiCacheService",
		func(s grpc.ServiceRegistrar) { pb.RegisterGenAiCacheServiceServer(s, &pb.UnimplementedGenAiCacheServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGenAiCacheServiceHandlerServer(ctx, mux, &pb.UnimplementedGenAiCacheServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.VertexRagDataService",
		func(s grpc.ServiceRegistrar) { pb.RegisterVertexRagDataServiceServer(s, &pb.UnimplementedVertexRagDataServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterVertexRagDataServiceHandlerServer(ctx, mux, &pb.UnimplementedVertexRagDataServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.DeploymentResourcePoolService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDeploymentResourcePoolServiceServer(s, &pb.UnimplementedDeploymentResourcePoolServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDeploymentResourcePoolServiceHandlerServer(ctx, mux, &pb.UnimplementedDeploymentResourcePoolServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.FeaturestoreService",
		func(s grpc.ServiceRegistrar) { pb.RegisterFeaturestoreServiceServer(s, &pb.UnimplementedFeaturestoreServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterFeaturestoreServiceHandlerServer(ctx, mux, &pb.UnimplementedFeaturestoreServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.PredictionService",
		func(s grpc.ServiceRegistrar) { pb.RegisterPredictionServiceServer(s, &pb.UnimplementedPredictionServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterPredictionServiceHandlerServer(ctx, mux, &pb.UnimplementedPredictionServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.ReasoningEngineExecutionService",
		func(s grpc.ServiceRegistrar) { pb.RegisterReasoningEngineExecutionServiceServer(s, &pb.UnimplementedReasoningEngineExecutionServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterReasoningEngineExecutionServiceHandlerServer(ctx, mux, &pb.UnimplementedReasoningEngineExecutionServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.DataFoundryService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDataFoundryServiceServer(s, &pb.UnimplementedDataFoundryServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDataFoundryServiceHandlerServer(ctx, mux, &pb.UnimplementedDataFoundryServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.PipelineService",
		func(s grpc.ServiceRegistrar) { pb.RegisterPipelineServiceServer(s, &pb.UnimplementedPipelineServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterPipelineServiceHandlerServer(ctx, mux, &pb.UnimplementedPipelineServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.VertexRagService",
		func(s grpc.ServiceRegistrar) { pb.RegisterVertexRagServiceServer(s, &pb.UnimplementedVertexRagServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterVertexRagServiceHandlerServer(ctx, mux, &pb.UnimplementedVertexRagServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.MatchService",
		func(s grpc.ServiceRegistrar) { pb.RegisterMatchServiceServer(s, &pb.UnimplementedMatchServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterMatchServiceHandlerServer(ctx, mux, &pb.UnimplementedMatchServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.FeatureOnlineStoreService",
		func(s grpc.ServiceRegistrar) { pb.RegisterFeatureOnlineStoreServiceServer(s, &pb.UnimplementedFeatureOnlineStoreServiceServer{}) },
		nil,
	)
	registerEntry("google.cloud.aiplatform.v1.EndpointService",
		func(s grpc.ServiceRegistrar) { pb.RegisterEndpointServiceServer(s, &pb.UnimplementedEndpointServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterEndpointServiceHandlerServer(ctx, mux, &pb.UnimplementedEndpointServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.ModelGardenService",
		func(s grpc.ServiceRegistrar) { pb.RegisterModelGardenServiceServer(s, &pb.UnimplementedModelGardenServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterModelGardenServiceHandlerServer(ctx, mux, &pb.UnimplementedModelGardenServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.MetadataService",
		func(s grpc.ServiceRegistrar) { pb.RegisterMetadataServiceServer(s, &pb.UnimplementedMetadataServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterMetadataServiceHandlerServer(ctx, mux, &pb.UnimplementedMetadataServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.FeaturestoreOnlineServingService",
		func(s grpc.ServiceRegistrar) { pb.RegisterFeaturestoreOnlineServingServiceServer(s, &pb.UnimplementedFeaturestoreOnlineServingServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterFeaturestoreOnlineServingServiceHandlerServer(ctx, mux, &pb.UnimplementedFeaturestoreOnlineServingServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.IndexService",
		func(s grpc.ServiceRegistrar) { pb.RegisterIndexServiceServer(s, &pb.UnimplementedIndexServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterIndexServiceHandlerServer(ctx, mux, &pb.UnimplementedIndexServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.AgentAnomalyDetectionService",
		func(s grpc.ServiceRegistrar) { pb.RegisterAgentAnomalyDetectionServiceServer(s, &pb.UnimplementedAgentAnomalyDetectionServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAgentAnomalyDetectionServiceHandlerServer(ctx, mux, &pb.UnimplementedAgentAnomalyDetectionServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.MigrationService",
		func(s grpc.ServiceRegistrar) { pb.RegisterMigrationServiceServer(s, &pb.UnimplementedMigrationServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterMigrationServiceHandlerServer(ctx, mux, &pb.UnimplementedMigrationServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.NotebookService",
		func(s grpc.ServiceRegistrar) { pb.RegisterNotebookServiceServer(s, &pb.UnimplementedNotebookServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNotebookServiceHandlerServer(ctx, mux, &pb.UnimplementedNotebookServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.TensorboardService",
		func(s grpc.ServiceRegistrar) { pb.RegisterTensorboardServiceServer(s, &pb.UnimplementedTensorboardServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTensorboardServiceHandlerServer(ctx, mux, &pb.UnimplementedTensorboardServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.EvaluationService",
		func(s grpc.ServiceRegistrar) { pb.RegisterEvaluationServiceServer(s, &pb.UnimplementedEvaluationServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterEvaluationServiceHandlerServer(ctx, mux, &pb.UnimplementedEvaluationServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.SessionService",
		func(s grpc.ServiceRegistrar) { pb.RegisterSessionServiceServer(s, &pb.UnimplementedSessionServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSessionServiceHandlerServer(ctx, mux, &pb.UnimplementedSessionServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.LlmUtilityService",
		func(s grpc.ServiceRegistrar) { pb.RegisterLlmUtilityServiceServer(s, &pb.UnimplementedLlmUtilityServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterLlmUtilityServiceHandlerServer(ctx, mux, &pb.UnimplementedLlmUtilityServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.JobService",
		func(s grpc.ServiceRegistrar) { pb.RegisterJobServiceServer(s, &pb.UnimplementedJobServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterJobServiceHandlerServer(ctx, mux, &pb.UnimplementedJobServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.ReasoningEngineService",
		func(s grpc.ServiceRegistrar) { pb.RegisterReasoningEngineServiceServer(s, &pb.UnimplementedReasoningEngineServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterReasoningEngineServiceHandlerServer(ctx, mux, &pb.UnimplementedReasoningEngineServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.DatasetService",
		func(s grpc.ServiceRegistrar) { pb.RegisterDatasetServiceServer(s, &pb.UnimplementedDatasetServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDatasetServiceHandlerServer(ctx, mux, &pb.UnimplementedDatasetServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.ScheduleService",
		func(s grpc.ServiceRegistrar) { pb.RegisterScheduleServiceServer(s, &pb.UnimplementedScheduleServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterScheduleServiceHandlerServer(ctx, mux, &pb.UnimplementedScheduleServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.ModelService",
		func(s grpc.ServiceRegistrar) { pb.RegisterModelServiceServer(s, &pb.UnimplementedModelServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterModelServiceHandlerServer(ctx, mux, &pb.UnimplementedModelServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.SpecialistPoolService",
		func(s grpc.ServiceRegistrar) { pb.RegisterSpecialistPoolServiceServer(s, &pb.UnimplementedSpecialistPoolServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSpecialistPoolServiceHandlerServer(ctx, mux, &pb.UnimplementedSpecialistPoolServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.PersistentResourceService",
		func(s grpc.ServiceRegistrar) { pb.RegisterPersistentResourceServiceServer(s, &pb.UnimplementedPersistentResourceServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterPersistentResourceServiceHandlerServer(ctx, mux, &pb.UnimplementedPersistentResourceServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.FeatureOnlineStoreAdminService",
		func(s grpc.ServiceRegistrar) { pb.RegisterFeatureOnlineStoreAdminServiceServer(s, &pb.UnimplementedFeatureOnlineStoreAdminServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterFeatureOnlineStoreAdminServiceHandlerServer(ctx, mux, &pb.UnimplementedFeatureOnlineStoreAdminServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.GenAiTuningService",
		func(s grpc.ServiceRegistrar) { pb.RegisterGenAiTuningServiceServer(s, &pb.UnimplementedGenAiTuningServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGenAiTuningServiceHandlerServer(ctx, mux, &pb.UnimplementedGenAiTuningServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.VizierService",
		func(s grpc.ServiceRegistrar) { pb.RegisterVizierServiceServer(s, &pb.UnimplementedVizierServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterVizierServiceHandlerServer(ctx, mux, &pb.UnimplementedVizierServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.FeatureRegistryService",
		func(s grpc.ServiceRegistrar) { pb.RegisterFeatureRegistryServiceServer(s, &pb.UnimplementedFeatureRegistryServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterFeatureRegistryServiceHandlerServer(ctx, mux, &pb.UnimplementedFeatureRegistryServiceServer{}) },
	)
	registerEntry("google.cloud.aiplatform.v1.IndexEndpointService",
		func(s grpc.ServiceRegistrar) { pb.RegisterIndexEndpointServiceServer(s, &pb.UnimplementedIndexEndpointServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterIndexEndpointServiceHandlerServer(ctx, mux, &pb.UnimplementedIndexEndpointServiceServer{}) },
	)
}
