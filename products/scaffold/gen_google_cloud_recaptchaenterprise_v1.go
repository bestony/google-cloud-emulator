package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/recaptchaenterprise/v1"
)

func init() {
	registerEntry("google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterRecaptchaEnterpriseServiceServer(s, &pb.UnimplementedRecaptchaEnterpriseServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterRecaptchaEnterpriseServiceHandlerServer(ctx, mux, &pb.UnimplementedRecaptchaEnterpriseServiceServer{})
		},
	)
}
