#!/usr/bin/env python3
"""Generate scaffold registrations for all generated-but-not-implemented gRPC
services.

For every gen/**/ *_grpc.pb.go service, emits products/scaffold/gen_<dir>.go
which registers the service's Unimplemented server (gRPC) and, when a gateway
handler exists, its REST handlers. Hand-written products are excluded.

Usage: tools/scaffold.py
"""
import os
import re
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GEN = os.path.join(REPO, "gen")
OUT = os.path.join(REPO, "products", "scaffold")
MODULE = "github.com/bestony/google-cloud-emulator"

# Services already implemented by hand-written products (full proto service name).
EXCLUDE = {
    "google.cloud.secretmanager.v1.SecretManagerService",
    # Hand-written products/secretmanager also serves this LRO helper.
    "google.cloud.secretmanager.v1.SecretOperationsService",
    "google.pubsub.v1.Publisher",
    "google.pubsub.v1.Subscriber",
    "google.cloud.kms.v1.KeyManagementService",
    "google.cloud.run.v2.Services",
    "google.longrunning.Operations",
}

REGISTER_RE = re.compile(r"^func (Register\w+Server)\(", re.M)
HANDLER_RE = re.compile(r"^func (Register\w+HandlerServer)\(", re.M)
DESC_RE = re.compile(r'ServiceName:\s*"([^"]+)"', re.M)
PKG_RE = re.compile(r"^package\s+(\w+)", re.M)
UNIMPL_RE_TEMPLATE = "Unimplemented{service}Server"


def package_name(gendir):
    for fn in os.listdir(gendir):
        if fn.endswith("_grpc.pb.go"):
            with open(os.path.join(gendir, fn), encoding="utf-8") as f:
                return PKG_RE.search(f.read(2000)).group(1)
    return None


def main():
    os.makedirs(OUT, exist_ok=True)
    # Clear previously generated files.
    for fn in os.listdir(OUT):
        if fn.startswith("gen_") and fn.endswith(".go"):
            os.remove(os.path.join(OUT, fn))

    entries = []  # (gendir, pkg, [(svc, grpcReg, gwReg)])
    for dp, _, fns in os.walk(GEN):
        grpc_files = [f for f in fns if f.endswith("_grpc.pb.go")]
        if not grpc_files:
            continue
        gw_text = ""
        for f in fns:
            if f.endswith(".pb.gw.go"):
                with open(os.path.join(dp, f), encoding="utf-8") as fh:
                    gw_text += fh.read()
        services = []
        for gf in grpc_files:
            with open(os.path.join(dp, gf), encoding="utf-8") as fh:
                src = fh.read()
            regs = REGISTER_RE.findall(src)
            descs = DESC_RE.findall(src)
            for i, reg in enumerate(regs):
                svc = descs[i] if i < len(descs) else ""
                if svc in EXCLUDE:
                    continue
                # service name -> unimplemented struct name (strip Register/Server)
                short = reg[len("Register"):-len("Server")]
                gw = None
                for h in HANDLER_RE.findall(gw_text):
                    if h == f"Register{short}HandlerServer":
                        gw = h
                        break
                services.append((svc, short, reg, gw))
        if not services:
            continue
        pkg = package_name(dp)
        if not pkg:
            continue
        entries.append((dp, pkg, services))

    total = 0
    for dp, pkg, services in entries:
        rel = os.path.relpath(dp, GEN)
        slug = re.sub(r"[^a-zA-Z0-9]+", "_", rel).strip("_")
        out_path = os.path.join(OUT, f"gen_{slug}.go")
        has_gw = any(gw for _, _, _, gw in services)
        imports = [
            "package scaffold",
            "",
            "import (",
            '\t"google.golang.org/grpc"',
        ]
        if has_gw:
            imports += [
                '',
                '\t"context"',
                '',
                '\t"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"',
            ]
        imports += [
            "",
            f'\tpb "{MODULE}/gen/{rel}"',
            ")",
            "",
            "func init() {",
        ]
        lines = imports
        for svc, short, reg, gw in services:
            total += 1
            lines.append(f'\tregisterEntry("{svc}",')
            lines.append(f'\t\tfunc(s grpc.ServiceRegistrar) {{ pb.{reg}(s, &pb.Unimplemented{short}Server{{}}) }},')
            if gw:
                lines.append(f'\t\tfunc(ctx context.Context, mux *runtime.ServeMux) error {{ return pb.{gw}(ctx, mux, &pb.Unimplemented{short}Server{{}}) }},')
            else:
                lines.append('\t\tnil,')
            lines.append('\t)')
        lines.append("}")
        lines.append("")
        with open(out_path, "w", encoding="utf-8") as f:
            f.write("\n".join(lines))

    # Central registry file.
    with open(os.path.join(OUT, "registry.go"), "w", encoding="utf-8") as f:
        f.write('''// Package scaffold registers every generated gRPC service that has no
// hand-written implementation. All methods return codes.Unimplemented (501
// over REST), keeping unknown surfaces honest while making every product
// reachable.
package scaffold

import (
	"context"
	"errors"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type entry struct {
	name string
	grpc func(grpc.ServiceRegistrar)
	gw   func(context.Context, *runtime.ServeMux) error
}

var entries []entry
var seen = make(map[string]bool)

func registerEntry(name string, grpcReg func(grpc.ServiceRegistrar), gwReg func(context.Context, *runtime.ServeMux) error) {
	if name != "" {
		if seen[name] {
			return
		}
		seen[name] = true
	}
	entries = append(entries, entry{name: name, grpc: grpcReg, gw: gwReg})
}

// RegisterAll registers all scaffolded services. Gateway registration errors
// (e.g. route conflicts) are skipped so gRPC keeps working; they are returned
// joined for logging.
func RegisterAll(grpcServer *grpc.Server, gateway *runtime.ServeMux) error {
	var errs []error
	for _, e := range entries {
		if e.grpc != nil {
			e.grpc(grpcServer)
		}
		if e.gw != nil {
			if err := e.gw(context.Background(), gateway); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
''')

    print(f"scaffolded {total} services in {len(entries)} gen packages -> {OUT}")


if __name__ == "__main__":
    main()
