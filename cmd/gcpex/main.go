// Command gcpex runs the Google Cloud emulator: a single h2c endpoint that
// serves gRPC and REST for every emulated product.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	"github.com/bestony/google-cloud-emulator/core/iam"
	"github.com/bestony/google-cloud-emulator/core/lro"
	longrunningpb "github.com/bestony/google-cloud-emulator/gen/google/longrunning"
	"github.com/bestony/google-cloud-emulator/products/compute"
	"github.com/bestony/google-cloud-emulator/products/datastore"
	"github.com/bestony/google-cloud-emulator/products/dns"
	"github.com/bestony/google-cloud-emulator/products/firestore"
	"github.com/bestony/google-cloud-emulator/products/gcs"
	"github.com/bestony/google-cloud-emulator/products/gke"
	iamapi "github.com/bestony/google-cloud-emulator/products/iam"
	"github.com/bestony/google-cloud-emulator/products/kms"
	"github.com/bestony/google-cloud-emulator/products/pubsub"
	"github.com/bestony/google-cloud-emulator/products/run"
	"github.com/bestony/google-cloud-emulator/products/scaffold"
	"github.com/bestony/google-cloud-emulator/products/secretmanager"
	"github.com/bestony/google-cloud-emulator/products/sql"
)

func main() {
	addr := flag.String("addr", envOr("GCPEX_ADDR", ":8080"), "address to listen on")
	flag.Parse()

	iamStore := iam.NewStore()
	lroStore := lro.NewStore()
	smState := secretmanager.NewState(iamStore)
	dnsState := dns.NewState(iamStore)
	gcsState := gcs.NewState(iamStore)
	pubsubState := pubsub.NewState()
	kmsState := kms.NewState(lroStore)
	runState := run.NewState(lroStore, iamStore)
	computeState := compute.NewState()
	sqlState := sql.NewState()
	gkeState := gke.NewState()
	iamState := iamapi.NewState(iamStore)
	datastoreState := datastore.NewState()
	firestoreState := firestore.NewState()

	grpcServer := grpc.NewServer()
	gateway := runtime.NewServeMux()
	// Every generated service starts as an honest UNIMPLEMENTED stub.
	// Registered FIRST: grpc-gateway resolves duplicate REST patterns in
	// favor of the LAST registration, so the hand-written products below
	// win any path conflicts (e.g. GKE and AlloyDB share
	// /v1/projects/*/locations/*/clusters/*, which real GCP separates by
	// host; the emulator is single-host).
	if err := scaffold.RegisterAll(grpcServer, gateway); err != nil {
		fmt.Fprintf(os.Stderr, "gcpex: scaffold gateway conflicts: %v\n", err)
	}
	if err := secretmanager.Register(grpcServer, gateway, smState); err != nil {
		fatal(err)
	}
	if err := pubsub.Register(grpcServer, gateway, pubsubState); err != nil {
		fatal(err)
	}
	if err := kms.Register(grpcServer, gateway, kmsState); err != nil {
		fatal(err)
	}
	if err := run.Register(grpcServer, gateway, runState); err != nil {
		fatal(err)
	}
	if err := compute.Register(grpcServer, gateway, computeState); err != nil {
		fatal(err)
	}
	if err := sql.Register(grpcServer, gateway, sqlState); err != nil {
		fatal(err)
	}
	if err := gke.Register(grpcServer, gateway, gkeState); err != nil {
		fatal(err)
	}
	if err := iamapi.Register(grpcServer, gateway, iamState); err != nil {
		fatal(err)
	}
	if err := datastore.Register(grpcServer, gateway, datastoreState); err != nil {
		fatal(err)
	}
	if err := firestore.Register(grpcServer, gateway, firestoreState); err != nil {
		fatal(err)
	}
	// Shared google.longrunning.Operations service for all products.
	opsServer := lro.NewServer(lroStore)
	longrunningpb.RegisterOperationsServer(grpcServer, opsServer)
	if err := longrunningpb.RegisterOperationsHandlerServer(context.Background(), gateway, opsServer); err != nil {
		fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/dns/v1/", dns.NewHandler(dnsState))
	gcsHandler := gcs.NewHandler(gcsState)
	mux.Handle("/storage/v1/", gcsHandler)
	mux.Handle("/upload/storage/v1/", gcsHandler)
	mux.Handle("/", firestore.WrapGateway(gke.WrapGateway(gateway), firestoreState))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isGRPC(r) {
			grpcServer.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("gcpex listening on %s (gRPC + REST)\n", lis.Addr())
	h2s := &http2.Server{}
	if err := (&http.Server{Handler: h2c.NewHandler(handler, h2s)}).Serve(lis); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "gcpex:", err)
	os.Exit(1)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func isGRPC(r *http.Request) bool {
	return r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc")
}
