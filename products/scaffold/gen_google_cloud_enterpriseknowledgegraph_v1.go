package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/enterpriseknowledgegraph/v1"
)

func init() {
	registerEntry("google.cloud.enterpriseknowledgegraph.v1.EnterpriseKnowledgeGraphService",
		func(s grpc.ServiceRegistrar) { pb.RegisterEnterpriseKnowledgeGraphServiceServer(s, &pb.UnimplementedEnterpriseKnowledgeGraphServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterEnterpriseKnowledgeGraphServiceHandlerServer(ctx, mux, &pb.UnimplementedEnterpriseKnowledgeGraphServiceServer{}) },
	)
}
