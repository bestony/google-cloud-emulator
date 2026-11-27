package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/oracledatabase/v1"
)

func init() {
	registerEntry("google.cloud.oracledatabase.v1.OracleDatabase",
		func(s grpc.ServiceRegistrar) { pb.RegisterOracleDatabaseServer(s, &pb.UnimplementedOracleDatabaseServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterOracleDatabaseHandlerServer(ctx, mux, &pb.UnimplementedOracleDatabaseServer{}) },
	)
}
