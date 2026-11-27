package secretmanager

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/emptypb"

	iampb "github.com/bestony/google-cloud-emulator/gen/google/iam/v1"
	smpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/secretmanager/v1"
	"github.com/bestony/google-cloud-emulator/core/auth"
	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	"github.com/bestony/google-cloud-emulator/core/iam"
	"github.com/bestony/google-cloud-emulator/core/pagination"
)

// Server implements SecretManagerServiceServer.
type Server struct {
	smpb.UnimplementedSecretManagerServiceServer
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

func (s *Server) ListSecrets(ctx context.Context, req *smpb.ListSecretsRequest) (*smpb.ListSecretsResponse, error) {
	if req.Parent == "" {
		return nil, invalidArg("parent is required")
	}
	if !ValidParent(req.Parent) {
		return nil, invalidArg("invalid parent %q: must be projects/{project} or projects/{project}/locations/{location}", req.Parent)
	}
	if req.Filter != "" {
		return nil, invalidArg("filter is not supported by the emulator")
	}
	pageSize := pagination.EffectivePageSize(req.PageSize, 50)
	secrets, next, total, code := s.state.ListSecrets(req.Parent, pageSize, req.PageToken)
	if code != "" {
		return nil, invalidArg("invalid page token")
	}
	return &smpb.ListSecretsResponse{Secrets: secrets, NextPageToken: next, TotalSize: int32(total)}, nil
}

func (s *Server) CreateSecret(ctx context.Context, req *smpb.CreateSecretRequest) (*smpb.Secret, error) {
	if req.Parent == "" {
		return nil, invalidArg("parent is required")
	}
	if !ValidParent(req.Parent) {
		return nil, invalidArg("invalid parent %q", req.Parent)
	}
	if req.SecretId == "" {
		return nil, invalidArg("secret_id is required")
	}
	if !ValidSecretID(req.SecretId) {
		return nil, invalidArg("invalid secret_id %q: must be 1-255 characters of letters, digits, hyphen or underscore", req.SecretId)
	}
	if req.Secret == nil {
		return nil, invalidArg("secret is required")
	}
	secret, code := s.state.CreateSecret(req.Parent, req.SecretId, req.Secret)
	if code == "already-exists" {
		return nil, gcperr.GRPC(gcperr.AlreadyExists, "Secret %s already exists", SecretResourceName(req.Parent, req.SecretId))
	}
	return secret, nil
}

func (s *Server) AddSecretVersion(ctx context.Context, req *smpb.AddSecretVersionRequest) (*smpb.SecretVersion, error) {
	if req.Parent == "" {
		return nil, invalidArg("parent is required")
	}
	if !ParseSecretName(req.Parent) {
		return nil, invalidArg("invalid parent %q: must be a secret resource name", req.Parent)
	}
	if req.Payload == nil {
		return nil, invalidArg("payload is required")
	}
	v, code := s.state.AddVersion(req.Parent, req.Payload)
	switch code {
	case "":
		return v, nil
	case "not-found":
		return nil, notFound("Secret %s not found", req.Parent)
	case "checksum-mismatch":
		return nil, invalidArg("provided data_crc32c does not match the payload data")
	default:
		return nil, gcperr.GRPC(gcperr.Internal, "unexpected error")
	}
}

func (s *Server) GetSecret(ctx context.Context, req *smpb.GetSecretRequest) (*smpb.Secret, error) {
	if req.Name == "" {
		return nil, invalidArg("name is required")
	}
	if !ParseSecretName(req.Name) {
		return nil, invalidArg("invalid secret resource name %q", req.Name)
	}
	secret, ok := s.state.GetSecret(req.Name)
	if !ok {
		return nil, notFound("Secret %s not found", req.Name)
	}
	return secret, nil
}

func (s *Server) UpdateSecret(ctx context.Context, req *smpb.UpdateSecretRequest) (*smpb.Secret, error) {
	if req.Secret == nil || req.Secret.Name == "" {
		return nil, invalidArg("secret.name is required")
	}
	if !ParseSecretName(req.Secret.Name) {
		return nil, invalidArg("invalid secret resource name %q", req.Secret.Name)
	}
	maskPaths := map[string]bool{}
	for _, p := range req.UpdateMask.GetPaths() {
		maskPaths[p] = true
	}
	secret, code := s.state.UpdateSecret(req.Secret.Name, req.Secret, maskPaths)
	switch code {
	case "":
		return secret, nil
	case "not-found":
		return nil, notFound("Secret %s not found", req.Secret.Name)
	case "replication-immutable":
		return nil, invalidArg("replication is immutable and cannot be updated")
	default:
		return nil, gcperr.GRPC(gcperr.Internal, "unexpected error")
	}
}

func (s *Server) DeleteSecret(ctx context.Context, req *smpb.DeleteSecretRequest) (*emptypb.Empty, error) {
	if req.Name == "" {
		return nil, invalidArg("name is required")
	}
	if !ParseSecretName(req.Name) {
		return nil, invalidArg("invalid secret resource name %q", req.Name)
	}
	if req.Etag != "" {
		cur, ok := s.state.GetSecret(req.Name)
		if !ok {
			return nil, notFound("Secret %s not found", req.Name)
		}
		if cur.Etag != req.Etag {
			return nil, gcperr.GRPC(gcperr.FailedPrecondition, "etag mismatch for Secret %s", req.Name)
		}
	}
	if !s.state.DeleteSecret(req.Name) {
		return nil, notFound("Secret %s not found", req.Name)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) ListSecretVersions(ctx context.Context, req *smpb.ListSecretVersionsRequest) (*smpb.ListSecretVersionsResponse, error) {
	if req.Parent == "" {
		return nil, invalidArg("parent is required")
	}
	if !ParseSecretName(req.Parent) {
		return nil, invalidArg("invalid parent %q: must be a secret resource name", req.Parent)
	}
	pageSize := pagination.EffectivePageSize(req.PageSize, 50)
	versions, next, total, code := s.state.ListVersions(req.Parent, req.Filter, pageSize, req.PageToken)
	switch code {
	case "":
		return &smpb.ListSecretVersionsResponse{Versions: versions, NextPageToken: next, TotalSize: int32(total)}, nil
	case "not-found":
		return nil, notFound("Secret %s not found", req.Parent)
	case "invalid-filter":
		return nil, invalidArg("unsupported filter %q: only \"state=ENABLED|DISABLED|DESTROYED\" is supported", req.Filter)
	default:
		return nil, invalidArg("invalid page token")
	}
}

func (s *Server) GetSecretVersion(ctx context.Context, req *smpb.GetSecretVersionRequest) (*smpb.SecretVersion, error) {
	if req.Name == "" {
		return nil, invalidArg("name is required")
	}
	v, _, code := s.state.GetVersion(req.Name)
	switch code {
	case "":
		return v, nil
	case "invalid-name":
		return nil, invalidArg("invalid secret version resource name %q", req.Name)
	default:
		return nil, notFound("SecretVersion %s not found", req.Name)
	}
}

func (s *Server) AccessSecretVersion(ctx context.Context, req *smpb.AccessSecretVersionRequest) (*smpb.AccessSecretVersionResponse, error) {
	if req.Name == "" {
		return nil, invalidArg("name is required")
	}
	ver, _, acode := s.state.AccessVersion(req.Name)
	switch acode {
	case "":
	case "invalid-name":
		return nil, invalidArg("invalid secret version resource name %q", req.Name)
	case "destroyed":
		return nil, gcperr.GRPC(gcperr.FailedPrecondition, "SecretVersion %s has been destroyed", req.Name)
	case "disabled":
		return nil, gcperr.GRPC(gcperr.FailedPrecondition, "SecretVersion %s is disabled", req.Name)
	default:
		return nil, notFound("SecretVersion %s not found", req.Name)
	}
	crc := int64(ver.Checksum)
	secretName, _, _ := ParseVersionName(req.Name)
	return &smpb.AccessSecretVersionResponse{
		Name: fmt.Sprintf("%s/versions/%d", secretName, ver.Number),
		Payload: &smpb.SecretPayload{
			Data:         ver.Data,
			DataCrc32C:   &crc,
		},
	}, nil
}

func (s *Server) DisableSecretVersion(ctx context.Context, req *smpb.DisableSecretVersionRequest) (*smpb.SecretVersion, error) {
	return s.setVersionState(req.Name, smpb.SecretVersion_DISABLED)
}

func (s *Server) EnableSecretVersion(ctx context.Context, req *smpb.EnableSecretVersionRequest) (*smpb.SecretVersion, error) {
	return s.setVersionState(req.Name, smpb.SecretVersion_ENABLED)
}

func (s *Server) DestroySecretVersion(ctx context.Context, req *smpb.DestroySecretVersionRequest) (*smpb.SecretVersion, error) {
	if req.Name == "" {
		return nil, invalidArg("name is required")
	}
	v, code := s.state.SetVersionState(req.Name, 0, true)
	return versionResult(v, code, req.Name)
}

func (s *Server) setVersionState(name string, target smpb.SecretVersion_State) (*smpb.SecretVersion, error) {
	if name == "" {
		return nil, invalidArg("name is required")
	}
	v, code := s.state.SetVersionState(name, target, false)
	return versionResult(v, code, name)
}

func versionResult(v *smpb.SecretVersion, code, name string) (*smpb.SecretVersion, error) {
	switch code {
	case "":
		return v, nil
	case "invalid-name":
		return nil, invalidArg("invalid secret version resource name %q", name)
	case "destroyed":
		return nil, gcperr.GRPC(gcperr.FailedPrecondition, "SecretVersion %s has been destroyed", name)
	default:
		return nil, notFound("SecretVersion %s not found", name)
	}
}

// --- IAM ---

func (s *Server) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest) (*iampb.Policy, error) {
	if err := s.checkIAMResource(req.Resource, true); err != nil {
		return nil, err
	}
	p := s.state.IAMStore().Get(req.Resource)
	return &iampb.Policy{
		Version:  p.Version,
		Bindings: toIAMBindings(p.Bindings),
		Etag:     []byte(p.Etag),
	}, nil
}

func (s *Server) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest) (*iampb.Policy, error) {
	if req.Resource == "" {
		return nil, invalidArg("resource is required")
	}
	if err := s.checkIAMResource(req.Resource, true); err != nil {
		return nil, err
	}
	if req.Policy == nil {
		return nil, invalidArg("policy is required")
	}
	bindings := make([]iam.Binding, 0, len(req.Policy.Bindings))
	for _, b := range req.Policy.Bindings {
		if b.Condition != nil {
			return nil, gcperr.GRPC(gcperr.Unimplemented, "IAM conditions are not supported by the emulator")
		}
		bindings = append(bindings, iam.Binding{Role: b.Role, Members: append([]string(nil), b.Members...)})
	}
	p := s.state.IAMStore().Set(req.Resource, iam.Policy{Version: req.Policy.Version, Bindings: bindings})
	return &iampb.Policy{
		Version:  p.Version,
		Bindings: toIAMBindings(p.Bindings),
		Etag:     []byte(p.Etag),
	}, nil
}

func (s *Server) TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest) (*iampb.TestIamPermissionsResponse, error) {
	if req.Resource == "" {
		return nil, invalidArg("resource is required")
	}
	// Per the API contract, a missing secret yields an empty permission set,
	// not NOT_FOUND.
	if ParseSecretName(req.Resource) {
		if _, ok := s.state.GetSecret(req.Resource); !ok {
			return &iampb.TestIamPermissionsResponse{}, nil
		}
	} else if !ValidParent(req.Resource) {
		return nil, invalidArg("invalid resource %q", req.Resource)
	}
	caller := auth.FromContext(ctx)
	allowed := s.state.IAMStore().Test(req.Resource, req.Permissions, caller.Email)
	return &iampb.TestIamPermissionsResponse{Permissions: allowed}, nil
}

// checkIAMResource validates an IAM resource string. When requireExists is
// true and the resource names a secret, the secret must exist.
func (s *Server) checkIAMResource(resource string, requireExists bool) error {
	if resource == "" {
		return invalidArg("resource is required")
	}
	if ParseSecretName(resource) {
		if requireExists {
			if _, ok := s.state.GetSecret(resource); !ok {
				return notFound("Secret %s not found", resource)
			}
		}
		return nil
	}
	if ValidParent(resource) || strings.HasPrefix(resource, "projects/") {
		return nil
	}
	return invalidArg("invalid resource %q", resource)
}

func toIAMBindings(in []iam.Binding) []*iampb.Binding {
	out := make([]*iampb.Binding, 0, len(in))
	for _, b := range in {
		out = append(out, &iampb.Binding{Role: b.Role, Members: append([]string(nil), b.Members...)})
	}
	return out
}
