package datastore

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	dspb "github.com/bestony/google-cloud-emulator/gen/google/datastore/v1"
)

func testKey(project, kind, name string) *dspb.Key {
	return &dspb.Key{
		PartitionId: &dspb.PartitionId{ProjectId: project},
		Path:        []*dspb.Key_PathElement{{Kind: kind, IdType: &dspb.Key_PathElement_Name{Name: name}}},
	}
}

func strValue(s string) *dspb.Value {
	return &dspb.Value{ValueType: &dspb.Value_StringValue{StringValue: s}}
}
func intValue(i int64) *dspb.Value {
	return &dspb.Value{ValueType: &dspb.Value_IntegerValue{IntegerValue: i}}
}
func boolValue(b bool) *dspb.Value {
	return &dspb.Value{ValueType: &dspb.Value_BooleanValue{BooleanValue: b}}
}

func entity(key *dspb.Key, props map[string]*dspb.Value) *dspb.Entity {
	return &dspb.Entity{Key: key, Properties: props}
}

func TestCommitLookupUpdateDeleteAndAllocateIds(t *testing.T) {
	srv := NewServer(NewState())
	ctx := context.Background()
	key := testKey("demo", "Task", "task-1")
	insert := &dspb.Mutation{Operation: &dspb.Mutation_Insert{Insert: entity(key, map[string]*dspb.Value{
		"title": strValue("write tests"), "done": boolValue(false), "priority": intValue(7),
	})}}
	commit, err := srv.Commit(ctx, &dspb.CommitRequest{ProjectId: "demo", Mode: dspb.CommitRequest_NON_TRANSACTIONAL, Mutations: []*dspb.Mutation{insert}})
	if err != nil {
		t.Fatalf("Commit insert: %v", err)
	}
	if len(commit.MutationResults) != 1 || commit.MutationResults[0].Version == 0 || commit.MutationResults[0].CreateTime == nil {
		t.Fatalf("unexpected mutation results: %+v", commit.MutationResults)
	}

	lookup, err := srv.Lookup(ctx, &dspb.LookupRequest{ProjectId: "demo", Keys: []*dspb.Key{key}})
	if err != nil || len(lookup.Found) != 1 || len(lookup.Missing) != 0 {
		t.Fatalf("Lookup: %v %+v", err, lookup)
	}
	if got := lookup.Found[0].Entity.Properties["title"].GetStringValue(); got != "write tests" {
		t.Fatalf("title = %q", got)
	}

	update := entity(key, map[string]*dspb.Value{"done": boolValue(true)})
	if _, err := srv.Commit(ctx, &dspb.CommitRequest{ProjectId: "demo", Mode: dspb.CommitRequest_NON_TRANSACTIONAL, Mutations: []*dspb.Mutation{{Operation: &dspb.Mutation_Update{Update: update}, PropertyMask: &dspb.PropertyMask{Paths: []string{"done"}}}}}); err != nil {
		t.Fatalf("Commit update: %v", err)
	}
	lookup, _ = srv.Lookup(ctx, &dspb.LookupRequest{ProjectId: "demo", Keys: []*dspb.Key{key}})
	if !lookup.Found[0].Entity.Properties["done"].GetBooleanValue() || lookup.Found[0].Entity.Properties["title"].GetStringValue() != "write tests" {
		t.Fatalf("masked update incorrect: %+v", lookup.Found[0].Entity.Properties)
	}

	if _, err := srv.Commit(ctx, &dspb.CommitRequest{ProjectId: "demo", Mode: dspb.CommitRequest_NON_TRANSACTIONAL, Mutations: []*dspb.Mutation{{Operation: &dspb.Mutation_Delete{Delete: key}}}}); err != nil {
		t.Fatalf("Commit delete: %v", err)
	}
	lookup, _ = srv.Lookup(ctx, &dspb.LookupRequest{ProjectId: "demo", Keys: []*dspb.Key{key}})
	if len(lookup.Missing) != 1 {
		t.Fatalf("deleted entity still found: %+v", lookup)
	}

	incomplete := &dspb.Key{PartitionId: &dspb.PartitionId{ProjectId: "demo"}, Path: []*dspb.Key_PathElement{{Kind: "Task"}}}
	alloc, err := srv.AllocateIds(ctx, &dspb.AllocateIdsRequest{ProjectId: "demo", Keys: []*dspb.Key{incomplete}})
	if err != nil || len(alloc.Keys) != 1 || alloc.Keys[0].Path[0].GetId() == 0 {
		t.Fatalf("AllocateIds: %v %+v", err, alloc)
	}
}

func TestStructuredQueryFiltersOrderingAndLimit(t *testing.T) {
	srv := NewServer(NewState())
	ctx := context.Background()
	muts := []*dspb.Mutation{
		{Operation: &dspb.Mutation_Insert{Insert: entity(testKey("demo", "Task", "a"), map[string]*dspb.Value{"done": boolValue(false), "priority": intValue(1)})}},
		{Operation: &dspb.Mutation_Insert{Insert: entity(testKey("demo", "Task", "b"), map[string]*dspb.Value{"done": boolValue(false), "priority": intValue(9)})}},
		{Operation: &dspb.Mutation_Insert{Insert: entity(testKey("demo", "Task", "c"), map[string]*dspb.Value{"done": boolValue(true), "priority": intValue(10)})}},
	}
	if _, err := srv.Commit(ctx, &dspb.CommitRequest{ProjectId: "demo", Mode: dspb.CommitRequest_NON_TRANSACTIONAL, Mutations: muts}); err != nil {
		t.Fatal(err)
	}
	filter := &dspb.Filter{FilterType: &dspb.Filter_PropertyFilter{PropertyFilter: &dspb.PropertyFilter{
		Property: &dspb.PropertyReference{Name: "done"},
		Op:       dspb.PropertyFilter_EQUAL,
		Value:    boolValue(false),
	}}}
	resp, err := srv.RunQuery(ctx, &dspb.RunQueryRequest{ProjectId: "demo", QueryType: &dspb.RunQueryRequest_Query{Query: &dspb.Query{
		Kind:   []*dspb.KindExpression{{Name: "Task"}},
		Filter: filter,
		Order:  []*dspb.PropertyOrder{{Property: &dspb.PropertyReference{Name: "priority"}, Direction: dspb.PropertyOrder_DESCENDING}},
		Limit:  wrapperspb.Int32(1),
	}}})
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	batch := resp.GetBatch()
	if len(batch.GetEntityResults()) != 1 || batch.GetMoreResults() != dspb.QueryResultBatch_MORE_RESULTS_AFTER_LIMIT {
		t.Fatalf("unexpected batch: %+v", batch)
	}
	first := batch.EntityResults[0].Entity
	if first.Key.Path[0].GetName() != "b" || first.Properties["priority"].GetIntegerValue() != 9 {
		t.Fatalf("unexpected first result: %+v", first)
	}
	resp2, err := srv.RunQuery(ctx, &dspb.RunQueryRequest{ProjectId: "demo", QueryType: &dspb.RunQueryRequest_Query{Query: &dspb.Query{
		Kind: []*dspb.KindExpression{{Name: "Task"}}, Filter: filter,
		Order:       []*dspb.PropertyOrder{{Property: &dspb.PropertyReference{Name: "priority"}, Direction: dspb.PropertyOrder_DESCENDING}},
		StartCursor: batch.EndCursor,
	}}})
	if err != nil || len(resp2.Batch.EntityResults) != 1 || resp2.Batch.EntityResults[0].Entity.Key.Path[0].GetName() != "a" {
		t.Fatalf("cursor query: %v %+v", err, resp2)
	}
}

func TestTransactions(t *testing.T) {
	srv := NewServer(NewState())
	ctx := context.Background()
	begin, err := srv.BeginTransaction(ctx, &dspb.BeginTransactionRequest{ProjectId: "demo"})
	if err != nil || len(begin.Transaction) == 0 {
		t.Fatalf("BeginTransaction: %v %+v", err, begin)
	}
	key := testKey("demo", "Task", "tx")
	_, err = srv.Commit(ctx, &dspb.CommitRequest{
		ProjectId: "demo", Mode: dspb.CommitRequest_TRANSACTIONAL,
		TransactionSelector: &dspb.CommitRequest_Transaction{Transaction: begin.Transaction},
		Mutations:           []*dspb.Mutation{{Operation: &dspb.Mutation_Insert{Insert: entity(key, map[string]*dspb.Value{"title": strValue("tx")})}}},
	})
	if err != nil {
		t.Fatalf("transactional Commit: %v", err)
	}
	lookup, err := srv.Lookup(ctx, &dspb.LookupRequest{ProjectId: "demo", Keys: []*dspb.Key{key}})
	if err != nil || len(lookup.Found) != 1 {
		t.Fatalf("committed transaction not visible: %v %+v", err, lookup)
	}

	begin2, _ := srv.BeginTransaction(ctx, &dspb.BeginTransactionRequest{ProjectId: "demo"})
	if _, err := srv.Rollback(ctx, &dspb.RollbackRequest{ProjectId: "demo", Transaction: begin2.Transaction}); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	_, err = srv.Commit(ctx, &dspb.CommitRequest{ProjectId: "demo", Mode: dspb.CommitRequest_TRANSACTIONAL, TransactionSelector: &dspb.CommitRequest_Transaction{Transaction: begin2.Transaction}})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("commit rolled-back transaction: got %v", err)
	}
}
