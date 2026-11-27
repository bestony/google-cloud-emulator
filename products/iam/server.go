package iam

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bestony/google-cloud-emulator/core/auth"
	coreiam "github.com/bestony/google-cloud-emulator/core/iam"
	iampb "github.com/bestony/google-cloud-emulator/gen/google/iam/v1"
)

func invalidArg(format string, args ...any) error {
	return status.Errorf(codes.InvalidArgument, format, args...)
}

// normalizeResource strips the optional "//<host>/" prefix IAM resource
// names carry (e.g. "//secretmanager.googleapis.com/projects/p/secrets/s"),
// so policies key the same way as product-level IAM handlers.
func normalizeResource(resource string) string {
	if strings.HasPrefix(resource, "//") {
		rest := resource[2:]
		if i := strings.Index(rest, "/"); i >= 0 {
			return rest[i+1:]
		}
	}
	return resource
}

func toProto(p coreiam.Policy) *iampb.Policy {
	out := &iampb.Policy{
		Version: p.Version,
		Etag:    []byte(p.Etag),
	}
	for _, b := range p.Bindings {
		out.Bindings = append(out.Bindings, &iampb.Binding{
			Role:    b.Role,
			Members: append([]string(nil), b.Members...),
		})
	}
	return out
}

type server struct {
	iampb.UnimplementedIAMPolicyServer
	state *State
}

func (s *server) SetIamPolicy(_ context.Context, req *iampb.SetIamPolicyRequest) (*iampb.Policy, error) {
	if req.GetResource() == "" {
		return nil, invalidArg("resource is required")
	}
	if req.GetPolicy() == nil {
		return nil, invalidArg("policy is required")
	}
	protoPolicy := req.GetPolicy()
	if len(protoPolicy.GetAuditConfigs()) > 0 {
		return nil, status.Error(codes.Unimplemented, "audit configs are not supported by the emulator")
	}
	p := coreiam.Policy{Version: protoPolicy.GetVersion()}
	for _, b := range protoPolicy.GetBindings() {
		if b.GetCondition() != nil {
			return nil, status.Error(codes.Unimplemented, "IAM conditions are not supported by the emulator")
		}
		p.Bindings = append(p.Bindings, coreiam.Binding{
			Role:    b.GetRole(),
			Members: append([]string(nil), b.GetMembers()...),
		})
	}
	stored := s.state.store.Set(normalizeResource(req.GetResource()), p)
	return toProto(stored), nil
}

func (s *server) GetIamPolicy(_ context.Context, req *iampb.GetIamPolicyRequest) (*iampb.Policy, error) {
	if req.GetResource() == "" {
		return nil, invalidArg("resource is required")
	}
	return toProto(s.state.store.Get(normalizeResource(req.GetResource()))), nil
}

func (s *server) TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest) (*iampb.TestIamPermissionsResponse, error) {
	if req.GetResource() == "" {
		return nil, invalidArg("resource is required")
	}
	caller := auth.FromContext(ctx)
	allowed := s.state.store.Test(normalizeResource(req.GetResource()), req.GetPermissions(), caller.Email)
	return &iampb.TestIamPermissionsResponse{Permissions: allowed}, nil
}
