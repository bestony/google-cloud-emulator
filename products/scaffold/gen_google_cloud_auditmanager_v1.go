package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/auditmanager/v1"
)

func init() {
	registerEntry("google.cloud.auditmanager.v1.AuditManager",
		func(s grpc.ServiceRegistrar) { pb.RegisterAuditManagerServer(s, &pb.UnimplementedAuditManagerServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAuditManagerHandlerServer(ctx, mux, &pb.UnimplementedAuditManagerServer{}) },
	)
}
