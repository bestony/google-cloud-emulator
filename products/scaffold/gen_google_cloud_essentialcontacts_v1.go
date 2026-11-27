package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/essentialcontacts/v1"
)

func init() {
	registerEntry("google.cloud.essentialcontacts.v1.EssentialContactsService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterEssentialContactsServiceServer(s, &pb.UnimplementedEssentialContactsServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterEssentialContactsServiceHandlerServer(ctx, mux, &pb.UnimplementedEssentialContactsServiceServer{})
		},
	)
}
