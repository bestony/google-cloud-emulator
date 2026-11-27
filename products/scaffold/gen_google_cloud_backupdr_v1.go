package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/backupdr/v1"
)

func init() {
	registerEntry("google.cloud.backupdr.v1.BackupDrProtectionSummary",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterBackupDrProtectionSummaryServer(s, &pb.UnimplementedBackupDrProtectionSummaryServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterBackupDrProtectionSummaryHandlerServer(ctx, mux, &pb.UnimplementedBackupDrProtectionSummaryServer{})
		},
	)
	registerEntry("google.cloud.backupdr.v1.BackupDR",
		func(s grpc.ServiceRegistrar) { pb.RegisterBackupDRServer(s, &pb.UnimplementedBackupDRServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterBackupDRHandlerServer(ctx, mux, &pb.UnimplementedBackupDRServer{})
		},
	)
}
