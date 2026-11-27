package firestore

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	fspb "github.com/bestony/google-cloud-emulator/gen/google/firestore/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var firestoreJSONMarshal = protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: false}
var firestoreJSONUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: false}

func writeProtoJSON(w http.ResponseWriter, msg proto.Message) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	out, err := firestoreJSONMarshal.Marshal(msg)
	if err != nil {
		gcperr.WriteREST(w, gcperr.Internal, "marshal response: %v", err)
		return
	}
	_, _ = w.Write(out)
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		gcperr.WriteREST(w, gcperr.Internal, "%v", err)
		return
	}
	gcperr.WriteREST(w, gcperr.Code(st.Code()), "%s", st.Message())
}

func decodeProtoBody(r *http.Request, msg proto.Message) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	return firestoreJSONUnmarshal.Unmarshal(body, msg)
}

func maskFromQuery(values []string) *fspb.DocumentMask {
	if len(values) == 0 {
		return nil
	}
	var paths []string
	for _, value := range values {
		for _, path := range strings.Split(value, ",") {
			if path != "" {
				paths = append(paths, path)
			}
		}
	}
	if len(paths) == 0 {
		return nil
	}
	return &fspb.DocumentMask{FieldPaths: paths}
}

type httpQueryStream struct {
	ctx    context.Context
	w      http.ResponseWriter
	failed error
}

func (s *httpQueryStream) Send(resp *fspb.RunQueryResponse) error {
	if s.failed != nil {
		return s.failed
	}
	out, err := firestoreJSONMarshal.Marshal(resp)
	if err == nil {
		_, err = s.w.Write(append(out, '\n'))
	}
	s.failed = err
	return err
}
func (s *httpQueryStream) SetHeader(metadata.MD) error  { return nil }
func (s *httpQueryStream) SendHeader(metadata.MD) error { return nil }
func (s *httpQueryStream) SetTrailer(metadata.MD)       {}
func (s *httpQueryStream) Context() context.Context     { return s.ctx }
func (s *httpQueryStream) SendMsg(any) error            { return nil }
func (s *httpQueryStream) RecvMsg(any) error            { return nil }

type httpBatchGetStream struct {
	ctx    context.Context
	w      http.ResponseWriter
	failed error
}

func (s *httpBatchGetStream) Send(resp *fspb.BatchGetDocumentsResponse) error {
	if s.failed != nil {
		return s.failed
	}
	out, err := firestoreJSONMarshal.Marshal(resp)
	if err == nil {
		_, err = s.w.Write(append(out, '\n'))
	}
	s.failed = err
	return err
}
func (s *httpBatchGetStream) SetHeader(metadata.MD) error  { return nil }
func (s *httpBatchGetStream) SendHeader(metadata.MD) error { return nil }
func (s *httpBatchGetStream) SetTrailer(metadata.MD)       {}
func (s *httpBatchGetStream) Context() context.Context     { return s.ctx }
func (s *httpBatchGetStream) SendMsg(any) error            { return nil }
func (s *httpBatchGetStream) RecvMsg(any) error            { return nil }

func firestoreResource(path string) (resource string, ok bool) {
	if !strings.HasPrefix(path, "/v1/projects/") {
		return "", false
	}
	resource = strings.TrimPrefix(path, "/v1/")
	parts := strings.Split(resource, "/")
	if len(parts) < 5 || parts[0] != "projects" || parts[2] != "databases" || !strings.HasPrefix(parts[4], "documents") {
		return "", false
	}
	return resource, true
}

func splitFirestoreAction(resource string) (base, action string) {
	for _, candidate := range []string{"batchGet", "beginTransaction", "commit", "rollback", "runQuery", "runAggregationQuery", "listCollectionIds", "batchWrite"} {
		suffix := ":" + candidate
		if strings.HasSuffix(resource, suffix) {
			return strings.TrimSuffix(resource, suffix), candidate
		}
	}
	return resource, ""
}

func documentRootAndRest(resource string) (database, rest string, ok bool) {
	idx := strings.Index(resource, "/documents")
	if idx < 0 {
		return "", "", false
	}
	database = resource[:idx]
	rest = strings.Trim(strings.TrimPrefix(resource[idx:], "/documents"), "/")
	return database, rest, validDatabase(database)
}

// WrapGateway serves Firestore REST routes that do not have generated
// grpc-gateway bindings in this repository. Non-Firestore paths pass through.
func WrapGateway(next http.Handler, st *State) http.Handler {
	srv := NewServer(st)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resource, ok := firestoreResource(r.URL.Path)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		base, action := splitFirestoreAction(resource)
		database, rest, ok := documentRootAndRest(base)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()

		if action != "" {
			srv.handleAction(w, r, srv, database, base, action)
			return
		}

		segments := []string{}
		if rest != "" {
			segments = strings.Split(rest, "/")
		}
		if len(segments) == 0 {
			gcperr.WriteREST(w, gcperr.NotFound, "Firestore resource not found")
			return
		}

		// Odd path length addresses a collection; even path length addresses a document.
		if len(segments)%2 == 1 {
			parent := database + "/documents"
			if len(segments) > 1 {
				parent += "/" + strings.Join(segments[:len(segments)-1], "/")
			}
			collectionID := segments[len(segments)-1]
			switch r.Method {
			case http.MethodGet:
				pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
				resp, err := srv.ListDocuments(ctx, &fspb.ListDocumentsRequest{
					Parent: parent, CollectionId: collectionID, PageSize: int32(pageSize), PageToken: r.URL.Query().Get("pageToken"), Mask: maskFromQuery(r.URL.Query()["mask.fieldPaths"]),
				})
				if err != nil {
					writeGRPCError(w, err)
					return
				}
				writeProtoJSON(w, resp)
			case http.MethodPost:
				doc := &fspb.Document{}
				if err := decodeProtoBody(r, doc); err != nil {
					gcperr.WriteREST(w, gcperr.InvalidArgument, "invalid document JSON: %v", err)
					return
				}
				resp, err := srv.CreateDocument(ctx, &fspb.CreateDocumentRequest{
					Parent: parent, CollectionId: collectionID, DocumentId: r.URL.Query().Get("documentId"), Document: doc, Mask: maskFromQuery(r.URL.Query()["mask.fieldPaths"]),
				})
				if err != nil {
					writeGRPCError(w, err)
					return
				}
				writeProtoJSON(w, resp)
			default:
				gcperr.WriteREST(w, gcperr.Unimplemented, "method %s is not implemented for Firestore collections", r.Method)
			}
			return
		}

		name := database + "/documents/" + strings.Join(segments, "/")
		switch r.Method {
		case http.MethodGet:
			resp, err := srv.GetDocument(ctx, &fspb.GetDocumentRequest{Name: name, Mask: maskFromQuery(r.URL.Query()["mask.fieldPaths"])})
			if err != nil {
				writeGRPCError(w, err)
				return
			}
			writeProtoJSON(w, resp)
		case http.MethodPatch:
			doc := &fspb.Document{}
			if err := decodeProtoBody(r, doc); err != nil {
				gcperr.WriteREST(w, gcperr.InvalidArgument, "invalid document JSON: %v", err)
				return
			}
			doc.Name = name
			resp, err := srv.UpdateDocument(ctx, &fspb.UpdateDocumentRequest{Document: doc, UpdateMask: maskFromQuery(r.URL.Query()["updateMask.fieldPaths"])})
			if err != nil {
				writeGRPCError(w, err)
				return
			}
			writeProtoJSON(w, resp)
		case http.MethodDelete:
			_, err := srv.DeleteDocument(ctx, &fspb.DeleteDocumentRequest{Name: name})
			if err != nil {
				writeGRPCError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write([]byte("{}"))
		default:
			gcperr.WriteREST(w, gcperr.Unimplemented, "method %s is not implemented for Firestore documents", r.Method)
		}
	})
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request, srv *Server, database, parent, action string) {
	if r.Method != http.MethodPost {
		gcperr.WriteREST(w, gcperr.Unimplemented, "Firestore %s requires POST", action)
		return
	}
	ctx := r.Context()
	switch action {
	case "beginTransaction":
		req := &fspb.BeginTransactionRequest{}
		if err := decodeProtoBody(r, req); err != nil {
			gcperr.WriteREST(w, gcperr.InvalidArgument, "invalid JSON: %v", err)
			return
		}
		req.Database = database
		resp, err := srv.BeginTransaction(ctx, req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeProtoJSON(w, resp)
	case "commit":
		req := &fspb.CommitRequest{}
		if err := decodeProtoBody(r, req); err != nil {
			gcperr.WriteREST(w, gcperr.InvalidArgument, "invalid JSON: %v", err)
			return
		}
		req.Database = database
		resp, err := srv.Commit(ctx, req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeProtoJSON(w, resp)
	case "rollback":
		req := &fspb.RollbackRequest{}
		if err := decodeProtoBody(r, req); err != nil {
			gcperr.WriteREST(w, gcperr.InvalidArgument, "invalid JSON: %v", err)
			return
		}
		req.Database = database
		_, err := srv.Rollback(ctx, req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte("{}"))
	case "batchWrite":
		req := &fspb.BatchWriteRequest{}
		if err := decodeProtoBody(r, req); err != nil {
			gcperr.WriteREST(w, gcperr.InvalidArgument, "invalid JSON: %v", err)
			return
		}
		req.Database = database
		resp, err := srv.BatchWrite(ctx, req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeProtoJSON(w, resp)
	case "batchGet":
		req := &fspb.BatchGetDocumentsRequest{}
		if err := decodeProtoBody(r, req); err != nil {
			gcperr.WriteREST(w, gcperr.InvalidArgument, "invalid JSON: %v", err)
			return
		}
		req.Database = database
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := srv.BatchGetDocuments(req, &httpBatchGetStream{ctx: ctx, w: w}); err != nil {
			writeGRPCError(w, err)
			return
		}
	case "runQuery":
		req := &fspb.RunQueryRequest{}
		if err := decodeProtoBody(r, req); err != nil {
			gcperr.WriteREST(w, gcperr.InvalidArgument, "invalid JSON: %v", err)
			return
		}
		req.Parent = parent
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := srv.RunQuery(req, &httpQueryStream{ctx: ctx, w: w}); err != nil {
			writeGRPCError(w, err)
			return
		}
	case "listCollectionIds":
		req := &fspb.ListCollectionIdsRequest{}
		if err := decodeProtoBody(r, req); err != nil {
			gcperr.WriteREST(w, gcperr.InvalidArgument, "invalid JSON: %v", err)
			return
		}
		req.Parent = parent
		resp, err := srv.ListCollectionIds(ctx, req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeProtoJSON(w, resp)
	default:
		gcperr.WriteREST(w, gcperr.Unimplemented, "Firestore action %s is not implemented", action)
	}
}

// keep net/http/status imports honest for old generated grpc-gateway clients
var _ = codes.OK
