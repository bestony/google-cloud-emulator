package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/chronicle/v1"
)

func init() {
	registerEntry("google.cloud.chronicle.v1.NativeDashboardService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterNativeDashboardServiceServer(s, &pb.UnimplementedNativeDashboardServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterNativeDashboardServiceHandlerServer(ctx, mux, &pb.UnimplementedNativeDashboardServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.EntityService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterEntityServiceServer(s, &pb.UnimplementedEntityServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterEntityServiceHandlerServer(ctx, mux, &pb.UnimplementedEntityServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.InstanceService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterInstanceServiceServer(s, &pb.UnimplementedInstanceServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterInstanceServiceHandlerServer(ctx, mux, &pb.UnimplementedInstanceServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.DataTableService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataTableServiceServer(s, &pb.UnimplementedDataTableServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataTableServiceHandlerServer(ctx, mux, &pb.UnimplementedDataTableServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.DashboardQueryService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDashboardQueryServiceServer(s, &pb.UnimplementedDashboardQueryServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDashboardQueryServiceHandlerServer(ctx, mux, &pb.UnimplementedDashboardQueryServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.FindingsRefinementService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterFindingsRefinementServiceServer(s, &pb.UnimplementedFindingsRefinementServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterFindingsRefinementServiceHandlerServer(ctx, mux, &pb.UnimplementedFindingsRefinementServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.SearchService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSearchServiceServer(s, &pb.UnimplementedSearchServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSearchServiceHandlerServer(ctx, mux, &pb.UnimplementedSearchServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.RuleService",
		func(s grpc.ServiceRegistrar) { pb.RegisterRuleServiceServer(s, &pb.UnimplementedRuleServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterRuleServiceHandlerServer(ctx, mux, &pb.UnimplementedRuleServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.ReferenceListService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterReferenceListServiceServer(s, &pb.UnimplementedReferenceListServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterReferenceListServiceHandlerServer(ctx, mux, &pb.UnimplementedReferenceListServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.DashboardChartService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDashboardChartServiceServer(s, &pb.UnimplementedDashboardChartServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDashboardChartServiceHandlerServer(ctx, mux, &pb.UnimplementedDashboardChartServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.FeedsService",
		func(s grpc.ServiceRegistrar) { pb.RegisterFeedsServiceServer(s, &pb.UnimplementedFeedsServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterFeedsServiceHandlerServer(ctx, mux, &pb.UnimplementedFeedsServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.DataAccessControlService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDataAccessControlServiceServer(s, &pb.UnimplementedDataAccessControlServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDataAccessControlServiceHandlerServer(ctx, mux, &pb.UnimplementedDataAccessControlServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.FeaturedContentNativeDashboardService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterFeaturedContentNativeDashboardServiceServer(s, &pb.UnimplementedFeaturedContentNativeDashboardServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterFeaturedContentNativeDashboardServiceHandlerServer(ctx, mux, &pb.UnimplementedFeaturedContentNativeDashboardServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.BigQueryExportService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterBigQueryExportServiceServer(s, &pb.UnimplementedBigQueryExportServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterBigQueryExportServiceHandlerServer(ctx, mux, &pb.UnimplementedBigQueryExportServiceServer{})
		},
	)
	registerEntry("google.cloud.chronicle.v1.RuleExecutionErrorService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterRuleExecutionErrorServiceServer(s, &pb.UnimplementedRuleExecutionErrorServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterRuleExecutionErrorServiceHandlerServer(ctx, mux, &pb.UnimplementedRuleExecutionErrorServiceServer{})
		},
	)
}
