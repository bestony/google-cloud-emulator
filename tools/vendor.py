#!/usr/bin/env python3
"""Vendor googleapis proto files and regenerate Go code.

Usage:
  tools/vendor.py path/to/api.proto [more.proto ...]   # vendor these + deps
  tools/vendor.py --scan                                # discover stable v1 services

For each input proto (path relative to the googleapis checkout), the script
recursively copies its import closure into protos/, rewrites go_package to
point at gen/<dir>, and finally runs `buf generate`.
"""
import os
import re
import subprocess
import sys

GOOGLEAPIS = os.environ.get("GOOGLEAPIS", os.path.expanduser("~/googleapis"))
REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PROTOS = os.path.join(REPO, "protos")
GEN = os.path.join(REPO, "gen")
MODULE = "github.com/bestony/google-cloud-emulator"

IMPORT_RE = re.compile(r'^\s*import\s+(?:public\s+|weak\s+)?"([^"]+)"', re.M)
GOPKG_RE = re.compile(r'^\s*option\s+go_package\s*=\s*"([^"]+)";', re.M)
PKG_RE = re.compile(r'^\s*package\s+([\w.]+)\s*;', re.M)
EDITION_RE = re.compile(r'^\s*edition\s*=', re.M)
SERVICE_RE = re.compile(r'^\s*service\s+\w+\s*\{', re.M)

SKIP_PREFIXES = ("google/protobuf/",)
# Already vendored with a custom go_package; never overwrite.
NEVER_OVERWRITE = {"google/rpc/status.proto"}

# Protos whose Go package must come from a module already in go.mod's
# dependency graph (grpc-gateway runtime imports genproto httpbody), so we
# must not generate our own copy: it would double-register the proto file.
FORCE_GO_PACKAGE = {
    "google/api/httpbody.proto": "google.golang.org/genproto/googleapis/api/httpbody;httpbody",
}

GO_KEYWORDS = {
    "break", "case", "chan", "const", "continue", "default", "defer", "else",
    "fallthrough", "for", "func", "go", "goto", "if", "import", "interface{}",
    "map", "package", "range", "return", "select", "struct", "switch", "type",
    "var",
}


def proto_path(rel):
    return os.path.join(GOOGLEAPIS, rel)


def read(rel):
    with open(proto_path(rel), encoding="utf-8") as f:
        return f.read()


def existing_package_name(rel):
    """If gen/<dir> already has generated Go files, reuse their package name
    so new files in the same directory compile."""
    d = os.path.join(GEN, os.path.dirname(rel))
    if not os.path.isdir(d):
        return None
    for fn in os.listdir(d):
        if fn.endswith(".pb.go"):
            with open(os.path.join(d, fn), encoding="utf-8") as f:
                head = f.read(4000)
            m = re.search(r"^package\s+(\w+)", head, re.M)
            if m:
                return m.group(1)
    return None


def go_package_for(rel, src):
    if rel in FORCE_GO_PACKAGE:
        return FORCE_GO_PACKAGE[rel]
    pkg = existing_package_name(rel)
    if pkg:
        d = os.path.dirname(rel)
        return f"{MODULE}/gen/{d};{pkg}"
    m = GOPKG_RE.search(src)
    pkg = None
    if m:
        tail = m.group(1).split(";")[-1]
        pkg = tail or None
    if not pkg or not re.fullmatch(r"[a-zA-Z_]\w*", pkg):
        pm = PKG_RE.search(src)
        proto_pkg = pm.group(1) if pm else "unknown"
        pkg = re.sub(r"\W+", "_", proto_pkg.split(".")[-1])
    if not re.fullmatch(r"[a-zA-Z_]\w*", pkg) or pkg in GO_KEYWORDS:
        pkg = "pb" + pkg if re.fullmatch(r"[a-zA-Z_]\w*", pkg) else "pb"
    d = os.path.dirname(rel)
    return f"{MODULE}/gen/{d};{pkg}"


def closure(roots):
    seen = set()
    stack = list(roots)
    while stack:
        rel = stack.pop()
        if rel in seen or rel.startswith(SKIP_PREFIXES):
            continue
        if not os.path.exists(proto_path(rel)):
            print(f"WARN missing: {rel}", file=sys.stderr)
            continue
        seen.add(rel)
        src = read(rel)
        for imp in IMPORT_RE.findall(src):
            stack.append(imp)
    return sorted(seen)


def vendor(files):
    copied, skipped_edition, skipped_existing = [], [], []
    for rel in files:
        dest = os.path.join(PROTOS, rel)
        if rel in NEVER_OVERWRITE or os.path.exists(dest):
            skipped_existing.append(rel)
            continue
        src = read(rel)
        if EDITION_RE.search(src):
            skipped_edition.append(rel)
            continue
        src = GOPKG_RE.sub(f'option go_package = "{go_package_for(rel, src)}";', src, count=1)
        os.makedirs(os.path.dirname(dest), exist_ok=True)
        with open(dest, "w", encoding="utf-8") as f:
            f.write(src)
        copied.append(rel)
    return copied, skipped_edition, skipped_existing


def discover():
    """Stable v1 service definitions not yet vendored."""
    roots = []
    base = os.path.join(GOOGLEAPIS, "google")
    for dirpath, _, filenames in os.walk(base):
        for fn in filenames:
            if not fn.endswith(".proto"):
                continue
            full = os.path.join(dirpath, fn)
            rel = os.path.relpath(full, GOOGLEAPIS)
            parts = rel.split(os.sep)
            # stable v1 only, skip alpha/beta and already-vendored paths
            if len(parts) >= 4 and parts[1] == "cloud" and parts[3] == "v1" or parts[2] == "v1":
                try:
                    src = read(rel)
                except OSError:
                    continue
                if SERVICE_RE.search(src) and not EDITION_RE.search(src):
                    if not os.path.exists(os.path.join(PROTOS, rel)):
                        roots.append(rel)
    return sorted(roots)


def main():
    args = sys.argv[1:]
    if args == ["--scan"]:
        roots = discover()
        print(f"discovered {len(roots)} service protos")
        for r in roots:
            print(r)
        return
    if not args:
        print(__doc__)
        sys.exit(1)
    files = closure(args)
    copied, skipped_edition, skipped_existing = vendor(files)
    print(f"vendored {len(copied)} protos "
          f"({len(skipped_existing)} already present, {len(skipped_edition)} editions skipped)")
    for r in skipped_edition:
        print(f"  edition-skip: {r}", file=sys.stderr)
    print("next: run tools/generate.py to regenerate Go code")


if __name__ == "__main__":
    main()
