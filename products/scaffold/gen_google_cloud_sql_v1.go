package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/sql/v1"
)

func init() {
	registerEntry("google.cloud.sql.v1.SqlWorkloadCapturesService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlWorkloadCapturesServiceServer(s, &pb.UnimplementedSqlWorkloadCapturesServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSqlWorkloadCapturesServiceHandlerServer(ctx, mux, &pb.UnimplementedSqlWorkloadCapturesServiceServer{})
		},
	)
	registerEntry("google.cloud.sql.v1.SqlTiersService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlTiersServiceServer(s, &pb.UnimplementedSqlTiersServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSqlTiersServiceHandlerServer(ctx, mux, &pb.UnimplementedSqlTiersServiceServer{})
		},
	)
	registerEntry("google.cloud.sql.v1.SqlFeatureEligibilityService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlFeatureEligibilityServiceServer(s, &pb.UnimplementedSqlFeatureEligibilityServiceServer{})
		},
		nil,
	)
	registerEntry("google.cloud.sql.v1.SqlFlagsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlFlagsServiceServer(s, &pb.UnimplementedSqlFlagsServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSqlFlagsServiceHandlerServer(ctx, mux, &pb.UnimplementedSqlFlagsServiceServer{})
		},
	)
	registerEntry("google.cloud.sql.v1.SqlEventsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlEventsServiceServer(s, &pb.UnimplementedSqlEventsServiceServer{})
		},
		nil,
	)
	registerEntry("google.cloud.sql.v1.SqlConnectService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlConnectServiceServer(s, &pb.UnimplementedSqlConnectServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSqlConnectServiceHandlerServer(ctx, mux, &pb.UnimplementedSqlConnectServiceServer{})
		},
	)
	registerEntry("google.cloud.sql.v1.BlueGreenDeploymentsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterBlueGreenDeploymentsServiceServer(s, &pb.UnimplementedBlueGreenDeploymentsServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterBlueGreenDeploymentsServiceHandlerServer(ctx, mux, &pb.UnimplementedBlueGreenDeploymentsServiceServer{})
		},
	)
	registerEntry("google.cloud.sql.v1.SqlBackupsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlBackupsServiceServer(s, &pb.UnimplementedSqlBackupsServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSqlBackupsServiceHandlerServer(ctx, mux, &pb.UnimplementedSqlBackupsServiceServer{})
		},
	)
	registerEntry("google.cloud.sql.v1.SqlRegionsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlRegionsServiceServer(s, &pb.UnimplementedSqlRegionsServiceServer{})
		},
		nil,
	)
	registerEntry("google.cloud.sql.v1.SqlInstanceNamesService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlInstanceNamesServiceServer(s, &pb.UnimplementedSqlInstanceNamesServiceServer{})
		},
		nil,
	)
	registerEntry("google.cloud.sql.v1.SqlAvailableDatabaseVersionsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlAvailableDatabaseVersionsServiceServer(s, &pb.UnimplementedSqlAvailableDatabaseVersionsServiceServer{})
		},
		nil,
	)
	registerEntry("google.cloud.sql.v1.SqlSslCertsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlSslCertsServiceServer(s, &pb.UnimplementedSqlSslCertsServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSqlSslCertsServiceHandlerServer(ctx, mux, &pb.UnimplementedSqlSslCertsServiceServer{})
		},
	)
	registerEntry("google.cloud.sql.v1.SqlIamPoliciesService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSqlIamPoliciesServiceServer(s, &pb.UnimplementedSqlIamPoliciesServiceServer{})
		},
		nil,
	)
}
