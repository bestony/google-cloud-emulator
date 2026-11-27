#!/usr/bin/env python3
"""Generate Go code for vendored protos, grouped per directory with fallback.

Full pipeline per proto directory:
  1. buf generate --path <dir>            (go + grpc + gateway)
  2. on failure, per file: full plugins
  3. on gateway failure (client-streaming path params), go + grpc only

Usage: tools/generate.py [--jobs N]
"""
import concurrent.futures
import os
import subprocess
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PROTOS = os.path.join(REPO, "protos")
ENV = dict(os.environ)
ENV["PATH"] = ENV.get("PATH", "") + ":/usr/local/go/bin:" + os.path.expanduser("~/go/bin")

NOGW_TEMPLATE = """version: v2
plugins:
  - local: protoc-gen-go
    out: gen
    opt: paths=source_relative
  - local: protoc-gen-go-grpc
    out: gen
    opt: paths=source_relative
"""


def buf_generate(path, template=None):
    # buf v2 --path is relative to the workspace root (the repo), and the
    # module lives under protos/.
    if not path.startswith("protos/"):
        path = "protos/" + path
    cmd = ["buf", "generate"]
    if template:
        cmd += ["--template", template]
    cmd += ["--path", path]
    return subprocess.run(cmd, cwd=REPO, env=ENV,
                          stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)


# Vendored for import resolution only; generated code must use the genproto
# package instead (see FORCE_GO_PACKAGE in vendor.py), so never emit these.
EXCLUDE_GEN = {"google/api/httpbody.proto", "google/rpc/status.proto"}


def gen_file(rel, full=True):
    """rel is relative to protos/."""
    template = None
    if not full:
        template = os.path.join(REPO, ".buf.gen.nogw.yaml")
    return buf_generate(rel, template)


def gen_dir(d):
    """Generate one directory; returns (dir, failures)."""
    failures = []
    r = buf_generate(d)
    if r.returncode == 0:
        return d, failures
    # Per-file fallback.
    files = sorted(
        os.path.relpath(os.path.join(dp, f), PROTOS)
        for dp, _, fns in os.walk(os.path.join(PROTOS, d))
        for f in fns if f.endswith(".proto") and os.path.relpath(os.path.join(dp, f), PROTOS) not in EXCLUDE_GEN
    )
    for rel in files:
        r1 = gen_file(rel, full=True)
        if r1.returncode != 0:
            r2 = gen_file(rel, full=False)
            if r2.returncode != 0:
                failures.append((rel, (r2.stderr or "").strip().splitlines()[:2]))
    return d, failures


def main():
    jobs = 8
    if "--jobs" in sys.argv:
        jobs = int(sys.argv[sys.argv.index("--jobs") + 1])
    with open(os.path.join(REPO, ".buf.gen.nogw.yaml"), "w") as f:
        f.write(NOGW_TEMPLATE)

    dirs = set()
    for dp, _, fns in os.walk(PROTOS):
        if any(f.endswith(".proto") for f in fns):
            dirs.add(os.path.relpath(dp, PROTOS))
    dirs = sorted(dirs)
    print(f"generating {len(dirs)} proto directories", flush=True)

    all_failures = []
    done = 0
    with concurrent.futures.ThreadPoolExecutor(max_workers=jobs) as ex:
        futs = {ex.submit(gen_dir, d): d for d in dirs}
        for fut in concurrent.futures.as_completed(futs):
            d, failures = fut.result()
            done += 1
            all_failures.extend((d, f) for f in failures)
            if done % 25 == 0 or done == len(dirs):
                print(f"progress {done}/{len(dirs)}, failures so far: {len(all_failures)}", flush=True)

    # Drop any stale output for excluded protos.
    for rel in EXCLUDE_GEN:
        base = os.path.join(REPO, "gen", os.path.dirname(rel), os.path.basename(rel)[:-len(".proto")])
        for suffix in (".pb.go", ".pb.gw.go", "_grpc.pb.go"):
            p = base + suffix
            if os.path.exists(p):
                os.remove(p)
                print(f"removed excluded {p}")

    if all_failures:
        print("\nFAILED FILES:")
        for d, (rel, err) in all_failures:
            print(f"  {rel}: {'; '.join(err)}")
        sys.exit(1)
    print("all generated")


if __name__ == "__main__":
    main()
