package run

import (
	"context"
	"strings"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	runpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/run/v2"
	iampb "github.com/bestony/google-cloud-emulator/gen/google/iam/v1"
	longrunningpb "github.com/bestony/google-cloud-emulator/gen/google/longrunning"
	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	"github.com/bestony/google-cloud-emulator/core/iam"
	"github.com/bestony/google-cloud-emulator/core/lro"
	"github.com/bestony/google-cloud-emulator/core/pagination"
)

// Server implements google.cloud.run.v2.Services.
type Server struct {
	runpb.UnimplementedServicesServer
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
	case code == "already-exists":
		return gcperr.GRPC(gcperr.AlreadyExists, "resource already exists")
	case code == "invalid-parent" || code == "invalid-name" || code == "invalid-page-token":
		return gcperr.GRPC(gcperr.InvalidArgument, "%s", code)
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

func maskPaths(mask *fieldmaskpb.FieldMask) []string {
	if mask == nil {
		return nil
	}
	return mask.GetPaths()
}

// --- Services ---

func (s *Server) ListServices(ctx context.Context, req *runpb.ListServicesRequest) (*runpb.ListServicesResponse, error) {
	if req.GetParent() == "" {
		return nil, invalidArg("parent is required")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	services, next, code := s.state.ListServices(req.GetParent(), pageSize, req.GetPageToken())
	if code != "" {
		return nil, stateErr(code)
	}
	return &runpb.ListServicesResponse{Services: services, NextPageToken: next}, nil
}

func (s *Server) GetService(ctx context.Context, req *runpb.GetServiceRequest) (*runpb.Service, error) {
	svc := s.state.GetService(req.GetName())
	if svc == nil {
		return nil, notFound("service %q not found", req.GetName())
	}
	return svc, nil
}

func (s *Server) CreateService(ctx context.Context, req *runpb.CreateServiceRequest) (*longrunningpb.Operation, error) {
	if req.GetParent() == "" {
		return nil, invalidArg("parent is required")
	}
	if req.GetServiceId() == "" {
		return nil, invalidArg("service_id is required")
	}
	if req.GetService() == nil {
		return nil, invalidArg("service is required")
	}
	if req.GetValidateOnly() {
		return nil, invalidArg("validate_only is not supported by the emulator")
	}
	svc, code := s.state.CreateService(req.GetParent(), req.GetServiceId(), req.GetService())
	if code != "" {
		return nil, stateErr(code)
	}
	return s.startOperation(svc.Name, "CreateService", svc)
}

func (s *Server) UpdateService(ctx context.Context, req *runpb.UpdateServiceRequest) (*longrunningpb.Operation, error) {
	if req.GetService() == nil || req.GetService().GetName() == "" {
		return nil, invalidArg("service.name is required")
	}
	if req.GetValidateOnly() {
		return nil, invalidArg("validate_only is not supported by the emulator")
	}
	svc, code := s.state.UpdateService(req.GetService(), maskPaths(req.GetUpdateMask()), req.GetForceNewRevision())
	if code != "" {
		return nil, stateErr(code)
	}
	return s.startOperation(svc.Name, "UpdateService", svc)
}

func (s *Server) DeleteService(ctx context.Context, req *runpb.DeleteServiceRequest) (*longrunningpb.Operation, error) {
	if req.GetName() == "" {
		return nil, invalidArg("name is required")
	}
	if req.GetValidateOnly() {
		return nil, invalidArg("validate_only is not supported by the emulator")
	}
	deleted := s.state.DeleteService(req.GetName())
	if deleted == nil {
		return nil, notFound("service %q not found", req.GetName())
	}
	return s.startOperation(req.GetName(), "DeleteService", deleted)
}

// --- Revisions ---

func (s *Server) GetRevision(ctx context.Context, req *runpb.GetRevisionRequest) (*runpb.Revision, error) {
	rev := s.state.GetRevision(req.GetName())
	if rev == nil {
		return nil, notFound("revision %q not found", req.GetName())
	}
	return rev, nil
}

func (s *Server) ListRevisions(ctx context.Context, req *runpb.ListRevisionsRequest) (*runpb.ListRevisionsResponse, error) {
	if req.GetParent() == "" {
		return nil, invalidArg("parent is required")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	revs, next, code := s.state.ListRevisions(req.GetParent(), pageSize, req.GetPageToken())
	if code != "" {
		return nil, stateErr(code)
	}
	return &runpb.ListRevisionsResponse{Revisions: revs, NextPageToken: next}, nil
}

// --- IAM ---

func toIAMBindings(in []*iampb.Binding) []iam.Binding {
	if in == nil {
		return nil
	}
	out := make([]iam.Binding, 0, len(in))
	for _, b := range in {
		out = append(out, iam.Binding{Role: b.GetRole(), Members: append([]string(nil), b.GetMembers()...)})
	}
	return out
}

func fromIAMBindings(in []iam.Binding) []*iampb.Binding {
	if in == nil {
		return nil
	}
	out := make([]*iampb.Binding, 0, len(in))
	for _, b := range in {
		out = append(out, &iampb.Binding{Role: b.Role, Members: append([]string(nil), b.Members...)})
	}
	return out
}

func (s *Server) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest) (*iampb.Policy, error) {
	if req.GetResource() == "" {
		return nil, invalidArg("resource is required")
	}
	p := s.state.IAMStore().Get(req.GetResource())
	return &iampb.Policy{Version: p.Version, Bindings: fromIAMBindings(p.Bindings), Etag: []byte(p.Etag)}, nil
}

func (s *Server) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest) (*iampb.Policy, error) {
	if req.GetResource() == "" {
		return nil, invalidArg("resource is required")
	}
	if req.GetPolicy() == nil {
		return nil, invalidArg("policy is required")
	}
	p := s.state.IAMStore().Set(req.GetResource(), iam.Policy{Version: req.GetPolicy().GetVersion(), Bindings: toIAMBindings(req.GetPolicy().GetBindings())})
	return &iampb.Policy{Version: p.Version, Bindings: fromIAMBindings(p.Bindings), Etag: []byte(p.Etag)}, nil
}

func (s *Server) TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest) (*iampb.TestIamPermissionsResponse, error) {
	if req.GetResource() == "" {
		return nil, invalidArg("resource is required")
	}
	allowed := s.state.IAMStore().Test(req.GetResource(), req.GetPermissions(), "user:emulator@localhost")
	return &iampb.TestIamPermissionsResponse{Permissions: allowed}, nil
}
