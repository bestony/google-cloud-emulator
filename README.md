# google-cloud-emulator

A local emulator for Google Cloud APIs, written in Go. Inspired by [aws/aws-emulator](https://github.com/aws/aws-emulator): API shapes are driven by the official [googleapis](https://github.com/googleapis/googleapis) protobuf models, and handwritten code only implements state and business rules.

## Status: M3

Control-plane CRUD for eight hand-written products, plus **every other stable v1 Google Cloud API surface reachable** via auto-generated scaffolds. Pure in-memory state.

| Product | Protocol | Coverage |
|---|---|---|
| Secret Manager (`secretmanager.googleapis.com`) | gRPC + REST (grpc-gateway) | 15/17 RPCs: secrets CRUD, versions (add/get/list/access/enable/disable/destroy), IAM policy get/set/test. `RotateSecret` / `EnableManagedRotation` return UNIMPLEMENTED (they require Cloud SQL integration). |
| Cloud DNS (`dns.googleapis.com`) | REST | `projects.get`, managed zones CRUD + list + IAM verbs, changes create/get/list (DNS mutations are synchronous in the real API), rrsets get/list. dnsKeys / responsePolicies / policies / outboundEndpoints / locations return UNIMPLEMENTED. |
| Pub/Sub (`pubsub.googleapis.com`) | gRPC + REST (grpc-gateway) | Topics and subscriptions CRUD, list with pagination, `DetachSubscription`, server-assigned subscription names, `_deleted-topic_` semantics. Data plane (`Publish`/`Pull`/streaming) returns UNIMPLEMENTED. |
| Cloud KMS (`cloudkms.googleapis.com`) | gRPC + REST (grpc-gateway) | Key rings CRUD, crypto keys CRUD, crypto key versions (create/list/get/update state/primary/destroy/restore). `DeleteCryptoKey`/`DeleteCryptoKeyVersion` return long-running operations via the shared `google.longrunning.Operations` service. Crypto data plane (`Encrypt`/`Decrypt`/`Asymmetric*`/`Mac*`/import jobs) returns UNIMPLEMENTED. |
| Cloud Run v2 (`run.googleapis.com`) | gRPC + REST (grpc-gateway) | Services CRUD + revisions (template updates roll a new revision) + IAM policy verbs. Create/update/delete return LROs. Jobs/executions/instances/worker pools are not served. |
| Compute Engine (`compute.googleapis.com`) | gRPC + REST (grpc-gateway) | Instances, disks, networks, subnetworks, firewalls, addresses, images, snapshots, instance templates, health checks CRUD; zonal/regional/global operations get/list; aggregated lists; machine types/zones/regions return honest empty reference data. ~114 other compute services stay UNIMPLEMENTED. |
| Cloud SQL (`sqladmin.googleapis.com`) | gRPC + REST (grpc-gateway) | Instances CRUD (state RUNNABLE, ETag, self links), users/databases/backup runs subresources, operations get/list. Operations complete instantly as DONE. Flags/SSL certs/import-export stay UNIMPLEMENTED. |
| GKE (`container.googleapis.com`) | gRPC + REST (grpc-gateway) | ClusterManager clusters CRUD + update (desired master version, monitoring, addons, …) + operations get/list. Node pools, server config, JSON web keys and IP rotation stay UNIMPLEMENTED. |

Core framework: single h2c port serving gRPC and REST, host/path routing, accept-any Bearer auth, `google.rpc.Code` → REST error envelope, pagination tokens, IAM policy store.

### Bulk coverage (scaffold)

`tools/vendor.py` pulls the dependency closure of every stable `v1` service proto from [googleapis/googleapis](https://github.com/googleapis/googleapis) into `protos/`, `tools/generate.py` regenerates `gen/` per directory, and `tools/scaffold.py` emits `products/scaffold/`: one registration per generated gRPC service (442 services across 144 packages) that embeds the generated `Unimplemented*Server`, so gRPC returns `UNIMPLEMENTED` and REST returns 501 with the standard error envelope. The scaffold registers FIRST and the hand-written products after: grpc-gateway resolves duplicate REST patterns in favor of the last registration, so hand-written products win path conflicts. One known conflict: GKE and AlloyDB both use `/v1/projects/*/locations/*/clusters/*` (real GCP separates them by host); the emulator is single-host, so that path serves GKE. Two protos (`google/api/httpbody.proto`, `google/rpc/status.proto`) intentionally use the genproto Go packages to avoid proto-registration conflicts with grpc-gateway's runtime.

## Run

```bash
go build -o gcpex ./cmd/gcpex
./gcpex -addr :8080   # or GCPEX_ADDR=:8080
```

- REST: `http://localhost:8080/v1/projects/{p}/secrets` (Secret Manager), `http://localhost:8080/dns/v1/projects/{p}/managedZones` (Cloud DNS), `http://localhost:8080/v1/{parent=projects/*/topics/*}` (Pub/Sub), `http://localhost:8080/v1/{parent=projects/*/locations/*}/keyRings` (KMS), `http://localhost:8080/v2/{parent=projects/*/locations/*}/services` (Cloud Run)
- gRPC: plaintext h2c on the same port (Secret Manager, Pub/Sub, KMS, Cloud Run Services and the shared `google.longrunning.Operations` service)
- `GET /healthz` → `ok`

Authentication accepts any Bearer token; the caller identity defaults to `dev@localhost`.

## Point a client at it

```bash
# Secret Manager (Go SDK)
client, _ := secretmanager.NewClient(ctx, option.WithEndpoint("localhost:8080"), option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))

# REST / curl
curl -X POST "http://localhost:8080/v1/projects/demo/secrets?secretId=my-secret" -d '{}'

# Cloud DNS
curl -X POST "http://localhost:8080/dns/v1/projects/demo/managedZones" -d '{"name":"example-com","dnsName":"example.com."}'
```

## Layout

```
cmd/gcpex          single-binary entrypoint (gRPC + REST on one h2c port)
core/              auth, errors (google.rpc.Code envelope), iam, lro, pagination, engine
gen/               generated protobuf/gRPC/gateway code (do not edit)
protos/            vendored googleapis .proto files (go_package rewritten)
products/<name>/   one package per emulated product: state + handlers + registration
```

## Regenerating code

Requires `buf`, the local protoc plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`), and a clone of googleapis at `~/googleapis` (`GOOGLEAPIS` env var overrides):

```bash
go mod download
python3 tools/vendor.py --scan   # list stable v1 service protos not yet vendored
python3 tools/vendor.py <proto>… # vendor specific products (dependency closure)
python3 tools/generate.py --jobs 8   # regenerate gen/ (per-directory, gateway fallback)
python3 tools/scaffold.py            # regenerate products/scaffold registrations
go build ./... && go test ./...
```

## Roadmap

- M1 (done): Pub/Sub, Cloud KMS, Cloud Run v2 control planes
- M2 (done): bulk vendor + scaffold — all stable v1 control-plane surfaces reachable, UNIMPLEMENTED until hand-written
- M3 (done): Tier-1 hand-written control planes — Compute Engine, Cloud SQL, GKE
- M4: data planes (GCS objects, Pub/Sub messages, Firestore, Datastore, …) and more Tier-1 hand implementations
