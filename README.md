# google-cloud-emulator

A local emulator for Google Cloud APIs, written in Go. Inspired by [aws/aws-emulator](https://github.com/aws/aws-emulator): API shapes are driven by the official [googleapis](https://github.com/googleapis/googleapis) protobuf models, and handwritten code only implements state and business rules.

## Status: M1

Control-plane CRUD for five products, pure in-memory state:

| Product | Protocol | Coverage |
|---|---|---|
| Secret Manager (`secretmanager.googleapis.com`) | gRPC + REST (grpc-gateway) | 15/17 RPCs: secrets CRUD, versions (add/get/list/access/enable/disable/destroy), IAM policy get/set/test. `RotateSecret` / `EnableManagedRotation` return UNIMPLEMENTED (they require Cloud SQL integration). |
| Cloud DNS (`dns.googleapis.com`) | REST | `projects.get`, managed zones CRUD + list + IAM verbs, changes create/get/list (DNS mutations are synchronous in the real API), rrsets get/list. dnsKeys / responsePolicies / policies / outboundEndpoints / locations return UNIMPLEMENTED. |
| Pub/Sub (`pubsub.googleapis.com`) | gRPC + REST (grpc-gateway) | Topics and subscriptions CRUD, list with pagination, `DetachSubscription`, server-assigned subscription names, `_deleted-topic_` semantics. Data plane (`Publish`/`Pull`/streaming) returns UNIMPLEMENTED. |
| Cloud KMS (`cloudkms.googleapis.com`) | gRPC + REST (grpc-gateway) | Key rings CRUD, crypto keys CRUD, crypto key versions (create/list/get/update state/primary/destroy/restore). `DeleteCryptoKey`/`DeleteCryptoKeyVersion` return long-running operations via the shared `google.longrunning.Operations` service. Crypto data plane (`Encrypt`/`Decrypt`/`Asymmetric*`/`Mac*`/import jobs) returns UNIMPLEMENTED. |
| Cloud Run v2 (`run.googleapis.com`) | gRPC + REST (grpc-gateway) | Services CRUD + revisions (template updates roll a new revision) + IAM policy verbs. Create/update/delete return LROs. Jobs/executions/instances/worker pools are not served. |

Core framework: single h2c port serving gRPC and REST, host/path routing, accept-any Bearer auth, `google.rpc.Code` → REST error envelope, pagination tokens, IAM policy store.

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

Requires `buf` and the local protoc plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`):

```bash
buf generate   # reads buf.yaml/buf.gen.yaml, writes gen/
```

## Roadmap

- M1 (done): Pub/Sub, Cloud KMS, Cloud Run v2 control planes
- M2: `gen-scaffold` to auto-generate UNIMPLEMENTED stubs for the long tail of Google Cloud APIs
- M3: data planes (GCS objects, Pub/Sub messages, Firestore, …)
