package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/ftp/v1"
)

func init() {
	registerEntry("google.cloud.ftp.v1.CloudFtp",
		func(s grpc.ServiceRegistrar) { pb.RegisterCloudFtpServer(s, &pb.UnimplementedCloudFtpServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCloudFtpHandlerServer(ctx, mux, &pb.UnimplementedCloudFtpServer{}) },
	)
}
