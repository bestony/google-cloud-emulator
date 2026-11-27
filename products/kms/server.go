package kms

import (
	"context"
	"strings"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	"github.com/bestony/google-cloud-emulator/core/lro"
	"github.com/bestony/google-cloud-emulator/core/pagination"
	kmspb "github.com/bestony/google-cloud-emulator/gen/google/cloud/kms/v1"
	longrunningpb "github.com/bestony/google-cloud-emulator/gen/google/longrunning"
)

// Server implements google.cloud.kms.v1.KeyManagementService.
type Server struct {
	kmspb.UnimplementedKeyManagementServiceServer
	state *State
}

// NewServer returns a Server backed by state.
func NewServer(state *State) *Server {
	return &Server{state: state}
}

func invalidArg(format string, args ...any) error {
	return gcperr.GRPC(gcperr.InvalidArgument, format, args...)
}

func notFound(format string, args ...any) error {
	return gcperr.GRPC(gcperr.NotFound, format, args...)
}

func stateErr(code string) error {
	switch {
	case code == "not-found":
		return gcperr.GRPC(gcperr.NotFound, "resource not found")
	case code == "version-not-found":
		return gcperr.GRPC(gcperr.NotFound, "crypto key version not found")
	case code == "already-exists":
		return gcperr.GRPC(gcperr.AlreadyExists, "resource already exists")
	case code == "invalid-name" || code == "invalid-parent" || code == "invalid-purpose" || code == "invalid-state":
		return gcperr.GRPC(gcperr.InvalidArgument, "%s", code)
	case code == "already-destroyed":
		return gcperr.GRPC(gcperr.FailedPrecondition, "crypto key version is already destroyed or destruction is scheduled")
	case code == "version-destroyed":
		return gcperr.GRPC(gcperr.FailedPrecondition, "crypto key version is destroyed or destruction is scheduled")
	case code == "not-restore-scheduled":
		return gcperr.GRPC(gcperr.FailedPrecondition, "crypto key version is not scheduled for restore")
	case code == "invalid-page-token":
		return gcperr.GRPC(gcperr.InvalidArgument, "invalid page token")
	default:
		if strings.HasPrefix(code, "invalid-mask:") {
			return gcperr.GRPC(gcperr.InvalidArgument, "update mask path %q is not supported by the emulator", strings.TrimPrefix(code, "invalid-mask:"))
		}
		return gcperr.GRPC(gcperr.InvalidArgument, "%s", code)
	}
}

// operationPrefix extracts projects/{p}/locations/{l}/operations from a resource name.
func operationPrefix(resourceName string) string {
	parts := strings.Split(resourceName, "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "locations" && i >= 1 && parts[0] == "projects" {
			return "projects/" + parts[1] + "/locations/" + parts[i+1] + "/operations"
		}
	}
	return "projects/-/locations/-/operations"
}

func (s *Server) startOperation(resourceName, kind string, result any) (*longrunningpb.Operation, error) {
	op := s.state.LROStore().Start(operationPrefix(resourceName), kind)
	s.state.LROStore().Complete(op, result)
	return lro.ToProto(op)
}

// --- Key rings ---

func (s *Server) ListKeyRings(ctx context.Context, req *kmspb.ListKeyRingsRequest) (*kmspb.ListKeyRingsResponse, error) {
	if req.GetParent() == "" {
		return nil, invalidArg("parent is required")
	}
	if req.GetFilter() != "" {
		return nil, invalidArg("filter is not supported by the emulator")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	krs, next, total, code := s.state.ListKeyRings(req.GetParent(), pageSize, req.GetPageToken())
	if code != "" {
		return nil, stateErr(code)
	}
	return &kmspb.ListKeyRingsResponse{KeyRings: krs, NextPageToken: next, TotalSize: int32(total)}, nil
}

func (s *Server) GetKeyRing(ctx context.Context, req *kmspb.GetKeyRingRequest) (*kmspb.KeyRing, error) {
	kr := s.state.GetKeyRing(req.GetName())
	if kr == nil {
		return nil, notFound("key ring %q not found", req.GetName())
	}
	return kr, nil
}

func (s *Server) CreateKeyRing(ctx context.Context, req *kmspb.CreateKeyRingRequest) (*kmspb.KeyRing, error) {
	if req.GetParent() == "" {
		return nil, invalidArg("parent is required")
	}
	if req.GetKeyRingId() == "" {
		return nil, invalidArg("key_ring_id is required")
	}
	kr, code := s.state.CreateKeyRing(req.GetParent(), req.GetKeyRingId(), req.GetKeyRing())
	if code != "" {
		return nil, stateErr(code)
	}
	return kr, nil
}

// --- Crypto keys ---

func (s *Server) ListCryptoKeys(ctx context.Context, req *kmspb.ListCryptoKeysRequest) (*kmspb.ListCryptoKeysResponse, error) {
	if req.GetParent() == "" {
		return nil, invalidArg("parent is required")
	}
	if req.GetFilter() != "" {
		return nil, invalidArg("filter is not supported by the emulator")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	keys, next, total, code := s.state.ListCryptoKeys(req.GetParent(), pageSize, req.GetPageToken())
	if code != "" {
		return nil, stateErr(code)
	}
	return &kmspb.ListCryptoKeysResponse{CryptoKeys: keys, NextPageToken: next, TotalSize: int32(total)}, nil
}

func (s *Server) GetCryptoKey(ctx context.Context, req *kmspb.GetCryptoKeyRequest) (*kmspb.CryptoKey, error) {
	ck := s.state.GetCryptoKey(req.GetName())
	if ck == nil {
		return nil, notFound("crypto key %q not found", req.GetName())
	}
	return ck, nil
}

func (s *Server) CreateCryptoKey(ctx context.Context, req *kmspb.CreateCryptoKeyRequest) (*kmspb.CryptoKey, error) {
	if req.GetParent() == "" {
		return nil, invalidArg("parent is required")
	}
	if req.GetCryptoKeyId() == "" {
		return nil, invalidArg("crypto_key_id is required")
	}
	if req.GetCryptoKey() == nil {
		return nil, invalidArg("crypto_key is required")
	}
	ck, code := s.state.CreateCryptoKey(req.GetParent(), req.GetCryptoKeyId(), req.GetCryptoKey(), req.GetSkipInitialVersionCreation(), req.GetTrustedWrappingEnabled())
	if code != "" {
		return nil, stateErr(code)
	}
	return ck, nil
}

func maskPaths(mask *fieldmaskpb.FieldMask) []string {
	if mask == nil {
		return nil
	}
	return mask.GetPaths()
}

func (s *Server) UpdateCryptoKey(ctx context.Context, req *kmspb.UpdateCryptoKeyRequest) (*kmspb.CryptoKey, error) {
	if req.GetCryptoKey() == nil || req.GetCryptoKey().GetName() == "" {
		return nil, invalidArg("crypto_key.name is required")
	}
	ck, code := s.state.UpdateCryptoKey(req.GetCryptoKey(), maskPaths(req.GetUpdateMask()))
	if code != "" {
		return nil, stateErr(code)
	}
	return ck, nil
}

func (s *Server) DeleteCryptoKey(ctx context.Context, req *kmspb.DeleteCryptoKeyRequest) (*longrunningpb.Operation, error) {
	if req.GetName() == "" {
		return nil, invalidArg("name is required")
	}
	if s.state.GetCryptoKey(req.GetName()) == nil {
		return nil, notFound("crypto key %q not found", req.GetName())
	}
	deleted := s.state.DeleteCryptoKeyEntry(req.GetName())
	return s.startOperation(req.GetName(), "DeleteCryptoKey", deleted)
}

// --- Crypto key versions ---

func (s *Server) ListCryptoKeyVersions(ctx context.Context, req *kmspb.ListCryptoKeyVersionsRequest) (*kmspb.ListCryptoKeyVersionsResponse, error) {
	if req.GetParent() == "" {
		return nil, invalidArg("parent is required")
	}
	if req.GetFilter() != "" {
		return nil, invalidArg("filter is not supported by the emulator")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	versions, next, total, code := s.state.ListCryptoKeyVersions(req.GetParent(), pageSize, req.GetPageToken())
	if code != "" {
		return nil, stateErr(code)
	}
	return &kmspb.ListCryptoKeyVersionsResponse{CryptoKeyVersions: versions, NextPageToken: next, TotalSize: int32(total)}, nil
}

func (s *Server) GetCryptoKeyVersion(ctx context.Context, req *kmspb.GetCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	v := s.state.GetCryptoKeyVersion(req.GetName())
	if v == nil {
		return nil, notFound("crypto key version %q not found", req.GetName())
	}
	return v, nil
}

func (s *Server) CreateCryptoKeyVersion(ctx context.Context, req *kmspb.CreateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	if req.GetParent() == "" {
		return nil, invalidArg("parent is required")
	}
	if req.GetCryptoKeyVersion() == nil {
		return nil, invalidArg("crypto_key_version is required")
	}
	v, code := s.state.CreateCryptoKeyVersion(req.GetParent(), req.GetCryptoKeyVersion())
	if code != "" {
		return nil, stateErr(code)
	}
	return v, nil
}

func (s *Server) UpdateCryptoKeyVersion(ctx context.Context, req *kmspb.UpdateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	if req.GetCryptoKeyVersion() == nil || req.GetCryptoKeyVersion().GetName() == "" {
		return nil, invalidArg("crypto_key_version.name is required")
	}
	// Only the state field is mutable on a version.
	for _, p := range maskPaths(req.GetUpdateMask()) {
		if p != "state" {
			return nil, invalidArg("update mask path %q is not supported by the emulator", p)
		}
	}
	v, code := s.state.UpdateCryptoKeyVersionState(req.GetCryptoKeyVersion().GetName(), req.GetCryptoKeyVersion().GetState())
	if code != "" {
		return nil, stateErr(code)
	}
	return v, nil
}

func (s *Server) UpdateCryptoKeyPrimaryVersion(ctx context.Context, req *kmspb.UpdateCryptoKeyPrimaryVersionRequest) (*kmspb.CryptoKey, error) {
	if req.GetName() == "" {
		return nil, invalidArg("name is required")
	}
	if req.GetCryptoKeyVersionId() == "" {
		return nil, invalidArg("crypto_key_version_id is required")
	}
	ck, code := s.state.UpdateCryptoKeyPrimaryVersion(req.GetName(), req.GetCryptoKeyVersionId())
	if code != "" {
		return nil, stateErr(code)
	}
	return ck, nil
}

func (s *Server) DestroyCryptoKeyVersion(ctx context.Context, req *kmspb.DestroyCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	if req.GetName() == "" {
		return nil, invalidArg("name is required")
	}
	v, code := s.state.DestroyCryptoKeyVersion(req.GetName())
	if code != "" {
		return nil, stateErr(code)
	}
	return v, nil
}

func (s *Server) RestoreCryptoKeyVersion(ctx context.Context, req *kmspb.RestoreCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	if req.GetName() == "" {
		return nil, invalidArg("name is required")
	}
	v, code := s.state.RestoreCryptoKeyVersion(req.GetName())
	if code != "" {
		return nil, stateErr(code)
	}
	return v, nil
}

func (s *Server) DeleteCryptoKeyVersion(ctx context.Context, req *kmspb.DeleteCryptoKeyVersionRequest) (*longrunningpb.Operation, error) {
	if req.GetName() == "" {
		return nil, invalidArg("name is required")
	}
	v, code := s.state.DestroyCryptoKeyVersion(req.GetName())
	if code != "" {
		return nil, stateErr(code)
	}
	return s.startOperation(req.GetName(), "DeleteCryptoKeyVersion", v)
}
