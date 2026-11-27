package scaffold

import (
	"google.golang.org/grpc"

	pb "github.com/bestony/google-cloud-emulator/gen/google/firestore/v1"
)

func init() {
	registerEntry("google.firestore.v1.Firestore",
		func(s grpc.ServiceRegistrar) { pb.RegisterFirestoreServer(s, &pb.UnimplementedFirestoreServer{}) },
		nil,
	)
}
