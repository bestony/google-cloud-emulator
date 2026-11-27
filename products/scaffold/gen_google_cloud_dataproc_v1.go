package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/dataproc/v1"
)

func init() {
	registerEntry("google.cloud.dataproc.v1.NodeGroupController",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterNodeGroupControllerServer(s, &pb.UnimplementedNodeGroupControllerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterNodeGroupControllerHandlerServer(ctx, mux, &pb.UnimplementedNodeGroupControllerServer{})
		},
	)
	registerEntry("google.cloud.dataproc.v1.SessionTemplateController",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSessionTemplateControllerServer(s, &pb.UnimplementedSessionTemplateControllerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSessionTemplateControllerHandlerServer(ctx, mux, &pb.UnimplementedSessionTemplateControllerServer{})
		},
	)
	registerEntry("google.cloud.dataproc.v1.JobController",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterJobControllerServer(s, &pb.UnimplementedJobControllerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterJobControllerHandlerServer(ctx, mux, &pb.UnimplementedJobControllerServer{})
		},
	)
	registerEntry("google.cloud.dataproc.v1.BatchController",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterBatchControllerServer(s, &pb.UnimplementedBatchControllerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterBatchControllerHandlerServer(ctx, mux, &pb.UnimplementedBatchControllerServer{})
		},
	)
	registerEntry("google.cloud.dataproc.v1.ClusterController",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterClusterControllerServer(s, &pb.UnimplementedClusterControllerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterClusterControllerHandlerServer(ctx, mux, &pb.UnimplementedClusterControllerServer{})
		},
	)
	registerEntry("google.cloud.dataproc.v1.AutoscalingPolicyService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterAutoscalingPolicyServiceServer(s, &pb.UnimplementedAutoscalingPolicyServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAutoscalingPolicyServiceHandlerServer(ctx, mux, &pb.UnimplementedAutoscalingPolicyServiceServer{})
		},
	)
	registerEntry("google.cloud.dataproc.v1.WorkflowTemplateService",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterWorkflowTemplateServiceServer(s, &pb.UnimplementedWorkflowTemplateServiceServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterWorkflowTemplateServiceHandlerServer(ctx, mux, &pb.UnimplementedWorkflowTemplateServiceServer{})
		},
	)
	registerEntry("google.cloud.dataproc.v1.SessionController",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterSessionControllerServer(s, &pb.UnimplementedSessionControllerServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterSessionControllerHandlerServer(ctx, mux, &pb.UnimplementedSessionControllerServer{})
		},
	)
}
