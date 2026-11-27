package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/gkebackup/v1"
)

func init() {
	registerEntry("google.cloud.gkebackup.v1.BackupForGKE",
		func(s grpc.ServiceRegistrar) { pb.RegisterBackupForGKEServer(s, &pb.UnimplementedBackupForGKEServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterBackupForGKEHandlerServer(ctx, mux, &pb.UnimplementedBackupForGKEServer{}) },
	)
}
