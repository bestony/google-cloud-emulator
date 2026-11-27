package firestore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fspb "github.com/bestony/google-cloud-emulator/gen/google/firestore/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const testDatabase = "projects/demo/databases/(default)"

func fsString(s string) *fspb.Value {
	return &fspb.Value{ValueType: &fspb.Value_StringValue{StringValue: s}}
}
func fsInt(i int64) *fspb.Value {
	return &fspb.Value{ValueType: &fspb.Value_IntegerValue{IntegerValue: i}}
}
func fsBool(b bool) *fspb.Value {
	return &fspb.Value{ValueType: &fspb.Value_BooleanValue{BooleanValue: b}}
}

type queryStream struct {
	ctx       context.Context
	responses []*fspb.RunQueryResponse
}

func (s *queryStream) Send(resp *fspb.RunQueryResponse) error {
	s.responses = append(s.responses, resp)
	return nil
}
func (s *queryStream) SetHeader(metadata.MD) error  { return nil }
func (s *queryStream) SendHeader(metadata.MD) error { return nil }
func (s *queryStream) SetTrailer(metadata.MD)       {}
func (s *queryStream) Context() context.Context     { return s.ctx }
func (s *queryStream) SendMsg(any) error            { return nil }
func (s *queryStream) RecvMsg(any) error            { return nil }

type batchGetStream struct {
	ctx       context.Context
	responses []*fspb.BatchGetDocumentsResponse
}

func (s *batchGetStream) Send(resp *fspb.BatchGetDocumentsResponse) error {
	s.responses = append(s.responses, resp)
	return nil
}
func (s *batchGetStream) SetHeader(metadata.MD) error  { return nil }
func (s *batchGetStream) SendHeader(metadata.MD) error { return nil }
func (s *batchGetStream) SetTrailer(metadata.MD)       {}
func (s *batchGetStream) Context() context.Context     { return s.ctx }
func (s *batchGetStream) SendMsg(any) error            { return nil }
func (s *batchGetStream) RecvMsg(any) error            { return nil }

func createUser(t *testing.T, srv *Server, id string, age int64, active bool) *fspb.Document {
	t.Helper()
	doc, err := srv.CreateDocument(context.Background(), &fspb.CreateDocumentRequest{
		Parent: testDatabase + "/documents", CollectionId: "users", DocumentId: id,
		Document: &fspb.Document{Fields: map[string]*fspb.Value{"age": fsInt(age), "active": fsBool(active), "name": fsString(id)}},
	})
	if err != nil {
		t.Fatalf("CreateDocument %s: %v", id, err)
	}
	return doc
}

func TestDocumentCRUDAndList(t *testing.T) {
	srv := NewServer(NewState())
	ctx := context.Background()
	doc := createUser(t, srv, "alice", 31, true)
	wantName := testDatabase + "/documents/users/alice"
	if doc.Name != wantName || doc.CreateTime == nil || doc.UpdateTime == nil {
		t.Fatalf("unexpected created document: %+v", doc)
	}
	if _, err := srv.CreateDocument(ctx, &fspb.CreateDocumentRequest{Parent: testDatabase + "/documents", CollectionId: "users", DocumentId: "alice", Document: &fspb.Document{}}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate create: %v", err)
	}
	got, err := srv.GetDocument(ctx, &fspb.GetDocumentRequest{Name: wantName})
	if err != nil || got.Fields["age"].GetIntegerValue() != 31 {
		t.Fatalf("GetDocument: %v %+v", err, got)
	}
	doc.Fields = map[string]*fspb.Value{"age": fsInt(32)}
	if _, err := srv.UpdateDocument(ctx, &fspb.UpdateDocumentRequest{Document: doc, UpdateMask: &fspb.DocumentMask{FieldPaths: []string{"age"}}}); err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}
	got, _ = srv.GetDocument(ctx, &fspb.GetDocumentRequest{Name: wantName})
	if got.Fields["age"].GetIntegerValue() != 32 || !got.Fields["active"].GetBooleanValue() {
		t.Fatalf("mask merge incorrect: %+v", got.Fields)
	}
	list, err := srv.ListDocuments(ctx, &fspb.ListDocumentsRequest{Parent: testDatabase + "/documents", CollectionId: "users"})
	if err != nil || len(list.Documents) != 1 || list.Documents[0].Name != wantName {
		t.Fatalf("ListDocuments: %v %+v", err, list)
	}
	collections, err := srv.ListCollectionIds(ctx, &fspb.ListCollectionIdsRequest{Parent: testDatabase + "/documents"})
	if err != nil || len(collections.CollectionIds) != 1 || collections.CollectionIds[0] != "users" {
		t.Fatalf("ListCollectionIds: %v %+v", err, collections)
	}
	if _, err := srv.DeleteDocument(ctx, &fspb.DeleteDocumentRequest{Name: wantName}); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	if _, err := srv.GetDocument(ctx, &fspb.GetDocumentRequest{Name: wantName}); status.Code(err) != codes.NotFound {
		t.Fatalf("deleted get: %v", err)
	}
}

func TestRunQueryFilterOrderLimitAndBatchGet(t *testing.T) {
	srv := NewServer(NewState())
	createUser(t, srv, "alice", 31, true)
	createUser(t, srv, "bob", 40, true)
	createUser(t, srv, "carol", 20, false)
	q := &fspb.StructuredQuery{
		From: []*fspb.StructuredQuery_CollectionSelector{{CollectionId: "users"}},
		Where: &fspb.StructuredQuery_Filter{FilterType: &fspb.StructuredQuery_Filter_FieldFilter{FieldFilter: &fspb.StructuredQuery_FieldFilter{
			Field: &fspb.StructuredQuery_FieldReference{FieldPath: "active"}, Op: fspb.StructuredQuery_FieldFilter_EQUAL, Value: fsBool(true),
		}}},
		OrderBy: []*fspb.StructuredQuery_Order{{Field: &fspb.StructuredQuery_FieldReference{FieldPath: "age"}, Direction: fspb.StructuredQuery_DESCENDING}},
		Limit:   wrapperspb.Int32(1),
	}
	stream := &queryStream{ctx: context.Background()}
	if err := srv.RunQuery(&fspb.RunQueryRequest{Parent: testDatabase + "/documents", QueryType: &fspb.RunQueryRequest_StructuredQuery{StructuredQuery: q}}, stream); err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	if len(stream.responses) != 1 || stream.responses[0].Document.Name != testDatabase+"/documents/users/bob" {
		t.Fatalf("unexpected query responses: %+v", stream.responses)
	}
	batch := &batchGetStream{ctx: context.Background()}
	missing := testDatabase + "/documents/users/missing"
	if err := srv.BatchGetDocuments(&fspb.BatchGetDocumentsRequest{Database: testDatabase, Documents: []string{testDatabase + "/documents/users/alice", missing}}, batch); err != nil {
		t.Fatalf("BatchGetDocuments: %v", err)
	}
	if len(batch.responses) != 2 || batch.responses[0].GetFound() == nil || batch.responses[1].GetMissing() != missing {
		t.Fatalf("unexpected batch responses: %+v", batch.responses)
	}
}

func TestCommitTransactionsAndTransforms(t *testing.T) {
	srv := NewServer(NewState())
	ctx := context.Background()
	createUser(t, srv, "alice", 31, true)
	begin, err := srv.BeginTransaction(ctx, &fspb.BeginTransactionRequest{Database: testDatabase})
	if err != nil || len(begin.Transaction) == 0 {
		t.Fatalf("BeginTransaction: %v %+v", err, begin)
	}
	name := testDatabase + "/documents/users/alice"
	commit, err := srv.Commit(ctx, &fspb.CommitRequest{Database: testDatabase, Transaction: begin.Transaction, Writes: []*fspb.Write{
		{Operation: &fspb.Write_Update{Update: &fspb.Document{Name: name, Fields: map[string]*fspb.Value{"age": fsInt(32)}}}, UpdateMask: &fspb.DocumentMask{FieldPaths: []string{"age"}}, UpdateTransforms: []*fspb.DocumentTransform_FieldTransform{{FieldPath: "age", TransformType: &fspb.DocumentTransform_FieldTransform_Increment{Increment: fsInt(1)}}}},
		{Operation: &fspb.Write_Update{Update: &fspb.Document{Name: testDatabase + "/documents/users/dave", Fields: map[string]*fspb.Value{"age": fsInt(18)}}}},
	}})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if len(commit.WriteResults) != 2 || len(commit.WriteResults[0].TransformResults) != 1 {
		t.Fatalf("unexpected commit results: %+v", commit)
	}
	got, err := srv.GetDocument(ctx, &fspb.GetDocumentRequest{Name: name})
	if err != nil || got.Fields["age"].GetIntegerValue() != 33 {
		t.Fatalf("transformed document: %v %+v", err, got)
	}
	if _, err := srv.GetDocument(ctx, &fspb.GetDocumentRequest{Name: testDatabase + "/documents/users/dave"}); err != nil {
		t.Fatalf("transaction insert missing: %v", err)
	}

	begin2, _ := srv.BeginTransaction(ctx, &fspb.BeginTransactionRequest{Database: testDatabase})
	if _, err := srv.Rollback(ctx, &fspb.RollbackRequest{Database: testDatabase, Transaction: begin2.Transaction}); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	_, err = srv.Commit(ctx, &fspb.CommitRequest{Database: testDatabase, Transaction: begin2.Transaction})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("commit rolled-back transaction: %v", err)
	}
}

func TestRESTGatewayShim(t *testing.T) {
	st := NewState()
	ts := httptest.NewServer(WrapGateway(http.NotFoundHandler(), st))
	defer ts.Close()
	base := ts.URL + "/v1/projects/demo/databases/(default)"

	resp, err := http.Post(base+"/documents/users?documentId=rest1", "application/json", strings.NewReader(`{"fields":{"age":{"integerValue":"42"},"active":{"booleanValue":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("REST create status=%d body=%s", resp.StatusCode, body)
	}
	var created map[string]any
	if err := json.Unmarshal(body, &created); err != nil || created["name"] != testDatabase+"/documents/users/rest1" {
		t.Fatalf("REST create body=%s err=%v", body, err)
	}

	resp, err = http.Get(base + "/documents/users/rest1")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "users/rest1") {
		t.Fatalf("REST get status=%d body=%s", resp.StatusCode, body)
	}

	req, _ := http.NewRequest(http.MethodPatch, base+"/documents/users/rest1?updateMask.fieldPaths=age", strings.NewReader(`{"fields":{"age":{"integerValue":"43"}}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"43"`) {
		t.Fatalf("REST patch status=%d body=%s", resp.StatusCode, body)
	}

	resp, err = http.Post(base+"/documents:runQuery", "application/json", strings.NewReader(`{"structuredQuery":{"from":[{"collectionId":"users"}],"where":{"fieldFilter":{"field":{"fieldPath":"age"},"op":"EQUAL","value":{"integerValue":"43"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "users/rest1") {
		t.Fatalf("REST runQuery status=%d body=%s", resp.StatusCode, body)
	}

	req, _ = http.NewRequest(http.MethodDelete, base+"/documents/users/rest1", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("REST delete status=%d body=%s", resp.StatusCode, body)
	}
}
