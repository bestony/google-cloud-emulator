package lro

import (
	"context"

	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	longrunningpb "github.com/bestony/google-cloud-emulator/gen/google/longrunning"
	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	"github.com/bestony/google-cloud-emulator/core/pagination"
)

// Server implements google.longrunning.Operations over a Store.
type Server struct {
	longrunningpb.UnimplementedOperationsServer
	store *Store
}

// NewServer returns an Operations server backed by store.
func NewServer(store *Store) *Server {
	return &Server{store: store}
}

func ToProto(op *Operation) (*longrunningpb.Operation, error) {
	out := &longrunningpb.Operation{
		Name: op.Name,
		Done: op.Done,
	}
	if op.Err != nil {
		out.Result = &longrunningpb.Operation_Error{Error: &status.Status{Code: int32(*op.Err), Message: op.Message}}
	} else if op.Done && op.Result != nil {
		if msg, ok := op.Result.(proto.Message); ok {
			if a, err := anypb.New(msg); err == nil {
				out.Result = &longrunningpb.Operation_Response{Response: a}
			}
		}
	}
	return out, nil
}

// GetOperation implements Operations.GetOperation.
func (s *Server) GetOperation(ctx context.Context, req *longrunningpb.GetOperationRequest) (*longrunningpb.Operation, error) {
	if req.GetName() == "" {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "name is required")
	}
	op, ok := s.store.Get(req.GetName())
	if !ok {
		return nil, gcperr.GRPC(gcperr.NotFound, "operation %q not found", req.GetName())
	}
	return ToProto(op)
}

// ListOperations implements Operations.ListOperations. req.Name is treated as
// a name prefix (the parent under which operations were started).
func (s *Server) ListOperations(ctx context.Context, req *longrunningpb.ListOperationsRequest) (*longrunningpb.ListOperationsResponse, error) {
	if req.GetFilter() != "" {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "filter is not supported by the emulator")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	all := s.store.List(req.GetName())
	// all is newest-first; apply offset pagination deterministically.
	offset, err := pagination.Decode(req.GetPageToken())
	if err != nil {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid page token")
	}
	total := len(all)
	resp := &longrunningpb.ListOperationsResponse{}
	if offset < total {
		end := offset + pageSize
		if end > total {
			end = total
		}
		for _, op := range all[offset:end] {
			p, err := ToProto(op)
			if err != nil {
				return nil, err
			}
			resp.Operations = append(resp.Operations, p)
		}
		if end < total {
			resp.NextPageToken = pagination.Encode(end)
		}
	}
	return resp, nil
}
