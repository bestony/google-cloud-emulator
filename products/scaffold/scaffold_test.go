package scaffold

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	compute "github.com/bestony/google-cloud-emulator/gen/google/cloud/compute/v1"
)

func setup(t *testing.T) (*grpc.Server, *runtime.ServeMux) {
	t.Helper()
	grpcServer := grpc.NewServer()
	gateway := runtime.NewServeMux()
	// Route conflicts between generated services are expected; gRPC
	// registration must still succeed for every service.
	_ = RegisterAll(grpcServer, gateway)
	return grpcServer, gateway
}

func TestRegisterAllGRPCUnimplemented(t *testing.T) {
	grpcServer, _ := setup(t)

	lis := bufconn.Listen(1 << 20)
	go func() { _ = grpcServer.Serve(lis) }()
	defer grpcServer.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Instances is hand-written and excluded from the scaffold; use
	// MachineImages, which is still scaffolded.
	client := compute.NewMachineImagesClient(conn)
	_, err = client.Get(context.Background(), &compute.GetMachineImageRequest{
		Project:      "test-project",
		MachineImage: "mi-1",
	})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("GetMachineImage: got %v, want Unimplemented", err)
	}
}

func TestRegisterAllRESTUnimplemented(t *testing.T) {
	_, gateway := setup(t)

	// Compute Engine's REST endpoint for an unimplemented method must return
	// an honest 501 rather than a fabricated resource.
	req := httptest.NewRequest(http.MethodGet,
		"https://compute.googleapis.com/compute/v1/projects/p/global/machineImages/i", nil)
	rec := httptest.NewRecorder()
	gateway.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		body := strings.TrimSpace(rec.Body.String())
		if len(body) > 200 {
			body = body[:200]
		}
		t.Fatalf("REST GetMachineImage: got status %d, want 501; body: %s", rec.Code, body)
	}
}
