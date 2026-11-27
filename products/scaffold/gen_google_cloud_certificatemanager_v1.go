package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/certificatemanager/v1"
)

func init() {
	registerEntry("google.cloud.certificatemanager.v1.CertificateManager",
		func(s grpc.ServiceRegistrar) { pb.RegisterCertificateManagerServer(s, &pb.UnimplementedCertificateManagerServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCertificateManagerHandlerServer(ctx, mux, &pb.UnimplementedCertificateManagerServer{}) },
	)
}
