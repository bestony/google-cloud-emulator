// Command gcpex runs the Google Cloud emulator: a single h2c endpoint that
// serves gRPC and REST for every emulated product.
package main

import (
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
	"github.com/bestony/google-cloud-emulator/products/dns"
	"github.com/bestony/google-cloud-emulator/products/secretmanager"
)

func main() {
	addr := flag.String("addr", envOr("GCPEX_ADDR", ":8080"), "address to listen on")
	flag.Parse()

	iamStore := iam.NewStore()
	smState := secretmanager.NewState(iamStore)
	dnsState := dns.NewState(iamStore)

	grpcServer := grpc.NewServer()
	gateway := runtime.NewServeMux()
	if err := secretmanager.Register(grpcServer, gateway, smState); err != nil {
		fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/dns/v1/", dns.NewHandler(dnsState))
	mux.Handle("/", gateway)

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
