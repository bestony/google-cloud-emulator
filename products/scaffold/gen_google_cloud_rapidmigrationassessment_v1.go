package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/rapidmigrationassessment/v1"
)

func init() {
	registerEntry("google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterRapidMigrationAssessmentServer(s, &pb.UnimplementedRapidMigrationAssessmentServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterRapidMigrationAssessmentHandlerServer(ctx, mux, &pb.UnimplementedRapidMigrationAssessmentServer{})
		},
	)
}
