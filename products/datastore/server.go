// Package datastore emulates the google.datastore.v1 data plane in memory.
package datastore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	dspb "github.com/bestony/google-cloud-emulator/gen/google/datastore/v1"
)

type entityRecord struct {
	entity     *dspb.Entity
	version    int64
	createTime *timestamppb.Timestamp
	updateTime *timestamppb.Timestamp
}

type transaction struct {
	data   map[string]map[string]*entityRecord
	active bool
}

// State holds entities, versions and transactions. The outer map key is
// project/database/namespace; the inner key is the canonical entity path.
type State struct {
	mu           sync.RWMutex
	entities     map[string]map[string]*entityRecord
	transactions map[string]*transaction
	nextVersion  int64
	nextID       int64
}

func NewState() *State {
	return &State{
		entities:     map[string]map[string]*entityRecord{},
		transactions: map[string]*transaction{},
		nextVersion:  1,
		nextID:       1000,
	}
}

// Server implements DatastoreServer.
type Server struct {
	dspb.UnimplementedDatastoreServer
	state *State
}

func NewServer(st *State) *Server {
	if st == nil {
		st = NewState()
	}
	return &Server{state: st}
}

func scopeFor(project, database, namespace string) string {
	return project + "\x00" + database + "\x00" + namespace
}

func normalizeKey(key *dspb.Key, project, database, namespace string) *dspb.Key {
	if key == nil {
		return nil
	}
	out := proto.Clone(key).(*dspb.Key)
	if out.PartitionId == nil {
		out.PartitionId = &dspb.PartitionId{}
	}
	if out.PartitionId.ProjectId == "" {
		out.PartitionId.ProjectId = project
	}
	if out.PartitionId.DatabaseId == "" {
		out.PartitionId.DatabaseId = database
	}
	if out.PartitionId.NamespaceId == "" {
		out.PartitionId.NamespaceId = namespace
	}
	return out
}

func keyParts(key *dspb.Key) (scope, path string, ok bool) {
	if key == nil || key.PartitionId == nil || len(key.Path) == 0 {
		return "", "", false
	}
	parts := make([]string, 0, len(key.Path))
	for _, el := range key.Path {
		if el.GetKind() == "" {
			return "", "", false
		}
		id := ""
		if el.GetName() != "" {
			id = "n:" + el.GetName()
		} else if el.GetId() != 0 {
			id = "i:" + strconv.FormatInt(el.GetId(), 10)
		} else {
			return "", "", false
		}
		parts = append(parts, el.GetKind()+":"+id)
	}
	p := key.PartitionId
	return scopeFor(p.ProjectId, p.DatabaseId, p.NamespaceId), strings.Join(parts, "\x00"), true
}

func canonicalKey(key *dspb.Key) (string, string, bool) {
	scope, path, ok := keyParts(key)
	return scope, scope + "\x00" + path, ok
}

func keyKind(key *dspb.Key) string {
	if key == nil || len(key.Path) == 0 {
		return ""
	}
	return key.Path[len(key.Path)-1].GetKind()
}

func cloneEntity(in *dspb.Entity) *dspb.Entity {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*dspb.Entity)
}

func cloneRecord(in *entityRecord) *entityRecord {
	if in == nil {
		return nil
	}
	out := *in
	out.entity = cloneEntity(in.entity)
	out.createTime = cloneTime(in.createTime)
	out.updateTime = cloneTime(in.updateTime)
	return &out
}

func cloneTime(in *timestamppb.Timestamp) *timestamppb.Timestamp {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*timestamppb.Timestamp)
}

func cloneData(in map[string]map[string]*entityRecord) map[string]map[string]*entityRecord {
	out := make(map[string]map[string]*entityRecord, len(in))
	for scope, rows := range in {
		out[scope] = make(map[string]*entityRecord, len(rows))
		for key, rec := range rows {
			out[scope][key] = cloneRecord(rec)
		}
	}
	return out
}

func (s *State) dataForRead(transactionID []byte) (map[string]map[string]*entityRecord, error) {
	if len(transactionID) == 0 {
		return s.entities, nil
	}
	tx, ok := s.transactions[string(transactionID)]
	if !ok || !tx.active {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "transaction is not active")
	}
	return tx.data, nil
}

func newTransactionID() []byte {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return []byte(hex.EncodeToString(buf))
}

func resultFor(rec *entityRecord, _ dspb.EntityResult_ResultType, projection []*dspb.Projection) *dspb.EntityResult {
	ent := cloneEntity(rec.entity)
	if len(projection) > 0 {
		keyOnly := false
		props := map[string]*dspb.Value{}
		for _, p := range projection {
			name := p.GetProperty().GetName()
			if name == "__key__" {
				keyOnly = true
				continue
			}
			if v, ok := ent.Properties[name]; ok {
				props[name] = proto.Clone(v).(*dspb.Value)
			}
		}
		if keyOnly && len(props) == 0 {
			ent.Properties = nil
		} else {
			ent.Properties = props
		}
	}
	return &dspb.EntityResult{Entity: ent, Version: rec.version, CreateTime: cloneTime(rec.createTime), UpdateTime: cloneTime(rec.updateTime)}
}

func (s *Server) Lookup(ctx context.Context, req *dspb.LookupRequest) (*dspb.LookupResponse, error) {
	if req.GetProjectId() == "" {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "project_id is required")
	}
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	data, err := s.state.dataForRead(req.GetReadOptions().GetTransaction())
	if err != nil {
		return nil, err
	}
	resp := &dspb.LookupResponse{ReadTime: timestamppb.Now()}
	for _, input := range req.GetKeys() {
		key := normalizeKey(input, req.GetProjectId(), req.GetDatabaseId(), "")
		scope, canonical, ok := canonicalKey(key)
		if !ok {
			return nil, gcperr.GRPC(gcperr.InvalidArgument, "key must have a complete path")
		}
		if rec, ok := data[scope][strings.TrimPrefix(canonical, scope+"\x00")]; ok {
			resp.Found = append(resp.Found, resultFor(rec, dspb.EntityResult_FULL, nil))
		} else {
			resp.Missing = append(resp.Missing, &dspb.EntityResult{Entity: &dspb.Entity{Key: key}, Version: 0})
		}
	}
	if req.GetReadOptions().GetNewTransaction() != nil {
		tx, err := s.beginTransactionLocked()
		if err != nil {
			return nil, err
		}
		resp.Transaction = tx
	}
	return resp, nil
}

func valueForProperty(ent *dspb.Entity, name string) (*dspb.Value, bool) {
	if name == "__key__" {
		return &dspb.Value{ValueType: &dspb.Value_KeyValue{KeyValue: ent.GetKey()}}, true
	}
	parts := strings.Split(name, ".")
	props := ent.GetProperties()
	var cur *dspb.Value
	for i, part := range parts {
		v, ok := props[part]
		if !ok {
			return nil, false
		}
		if i == len(parts)-1 {
			cur = v
			break
		}
		nested := v.GetEntityValue()
		if nested == nil {
			return nil, false
		}
		props = nested.GetProperties()
	}
	return cur, cur != nil
}

func valueRank(v *dspb.Value) int {
	switch v.GetValueType().(type) {
	case *dspb.Value_NullValue:
		return 0
	case *dspb.Value_BooleanValue:
		return 1
	case *dspb.Value_IntegerValue, *dspb.Value_DoubleValue:
		return 2
	case *dspb.Value_TimestampValue:
		return 3
	case *dspb.Value_KeyValue:
		return 4
	case *dspb.Value_StringValue:
		return 5
	case *dspb.Value_BlobValue:
		return 6
	case *dspb.Value_GeoPointValue:
		return 7
	case *dspb.Value_EntityValue:
		return 8
	case *dspb.Value_ArrayValue:
		return 9
	default:
		return 10
	}
}

func compareValues(a, b *dspb.Value) int {
	if a == nil || b == nil {
		if a == b {
			return 0
		}
		if a == nil {
			return -1
		}
		return 1
	}
	ra, rb := valueRank(a), valueRank(b)
	if ra != rb {
		if ra < rb {
			return -1
		}
		return 1
	}
	switch ra {
	case 1:
		av, bv := a.GetBooleanValue(), b.GetBooleanValue()
		if av == bv {
			return 0
		}
		if !av {
			return -1
		}
		return 1
	case 2:
		af, bf := numericValue(a), numericValue(b)
		if af == bf {
			return 0
		}
		if af < bf {
			return -1
		}
		return 1
	case 3:
		at, bt := a.GetTimestampValue().AsTime(), b.GetTimestampValue().AsTime()
		if at.Equal(bt) {
			return 0
		}
		if at.Before(bt) {
			return -1
		}
		return 1
	case 4:
		_, ak, _ := canonicalKey(a.GetKeyValue())
		_, bk, _ := canonicalKey(b.GetKeyValue())
		return strings.Compare(ak, bk)
	case 5:
		return strings.Compare(a.GetStringValue(), b.GetStringValue())
	case 6:
		return strings.Compare(string(a.GetBlobValue()), string(b.GetBlobValue()))
	default:
		if proto.Equal(a, b) {
			return 0
		}
		return strings.Compare(a.String(), b.String())
	}
}

func numericValue(v *dspb.Value) float64 {
	if iv, ok := v.GetValueType().(*dspb.Value_IntegerValue); ok {
		return float64(iv.IntegerValue)
	}
	return v.GetDoubleValue()
}

func equalValues(a, b *dspb.Value) bool { return compareValues(a, b) == 0 }

func matchesPropertyFilter(ent *dspb.Entity, pf *dspb.PropertyFilter) bool {
	if pf == nil {
		return true
	}
	name := pf.GetProperty().GetName()
	actual, exists := valueForProperty(ent, name)
	if pf.GetOp() == dspb.PropertyFilter_HAS_ANCESTOR {
		ancestor := pf.GetValue().GetKeyValue()
		if ancestor == nil || ent.GetKey() == nil {
			return false
		}
		_, av, aok := canonicalKey(ancestor)
		_, ek, eok := canonicalKey(ent.GetKey())
		return aok && eok && strings.HasPrefix(ek, av)
	}
	if !exists {
		return false
	}
	want := pf.GetValue()
	cmp := compareValues(actual, want)
	switch pf.GetOp() {
	case dspb.PropertyFilter_LESS_THAN:
		return cmp < 0
	case dspb.PropertyFilter_LESS_THAN_OR_EQUAL:
		return cmp <= 0
	case dspb.PropertyFilter_GREATER_THAN:
		return cmp > 0
	case dspb.PropertyFilter_GREATER_THAN_OR_EQUAL:
		return cmp >= 0
	case dspb.PropertyFilter_EQUAL:
		return cmp == 0
	case dspb.PropertyFilter_NOT_EQUAL:
		return cmp != 0
	case dspb.PropertyFilter_IN:
		for _, candidate := range want.GetArrayValue().GetValues() {
			if equalValues(actual, candidate) {
				return true
			}
		}
		return false
	case dspb.PropertyFilter_NOT_IN:
		for _, candidate := range want.GetArrayValue().GetValues() {
			if equalValues(actual, candidate) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func matchesFilter(ent *dspb.Entity, f *dspb.Filter) bool {
	if f == nil {
		return true
	}
	if pf := f.GetPropertyFilter(); pf != nil {
		return matchesPropertyFilter(ent, pf)
	}
	cf := f.GetCompositeFilter()
	if cf == nil {
		return true
	}
	if cf.GetOp() == dspb.CompositeFilter_OR {
		for _, child := range cf.GetFilters() {
			if matchesFilter(ent, child) {
				return true
			}
		}
		return false
	}
	for _, child := range cf.GetFilters() {
		if !matchesFilter(ent, child) {
			return false
		}
	}
	return true
}

func decodeCursor(cursor []byte) int {
	if len(cursor) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(string(cursor))
	if n < 0 {
		return 0
	}
	return n
}

func (s *Server) runQuery(data map[string]map[string]*entityRecord, req *dspb.RunQueryRequest) (*dspb.QueryResultBatch, error) {
	q := req.GetQuery()
	if q == nil {
		return nil, gcperr.GRPC(gcperr.Unimplemented, "GQL queries are not implemented; use structured queries")
	}
	partition := req.GetPartitionId()
	project, database, namespace := req.GetProjectId(), req.GetDatabaseId(), ""
	if partition != nil {
		if partition.GetProjectId() != "" {
			project = partition.GetProjectId()
		}
		if partition.GetDatabaseId() != "" {
			database = partition.GetDatabaseId()
		}
		namespace = partition.GetNamespaceId()
	}
	scope := scopeFor(project, database, namespace)
	rows := data[scope]
	var records []*entityRecord
	kinds := map[string]bool{}
	for _, kind := range q.GetKind() {
		kinds[kind.GetName()] = true
	}
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rec := rows[key]
		ent := rec.entity
		if len(kinds) > 0 && !kinds[keyKind(ent.GetKey())] {
			continue
		}
		if !matchesFilter(ent, q.GetFilter()) {
			continue
		}
		records = append(records, rec)
	}
	// Apply ordering. Missing properties sort before present properties.
	for i := len(q.GetOrder()) - 1; i >= 0; i-- {
		order := q.GetOrder()[i]
		sort.SliceStable(records, func(a, b int) bool {
			av, aok := valueForProperty(records[a].entity, order.GetProperty().GetName())
			bv, bok := valueForProperty(records[b].entity, order.GetProperty().GetName())
			if !aok && !bok {
				return false
			}
			if !aok {
				return true
			}
			if !bok {
				return false
			}
			cmp := compareValues(av, bv)
			if order.GetDirection() == dspb.PropertyOrder_DESCENDING {
				return cmp > 0
			}
			return cmp < 0
		})
	}
	start := decodeCursor(q.GetStartCursor()) + int(q.GetOffset())
	if start > len(records) {
		start = len(records)
	}
	end := len(records)
	limited := false
	if q.GetLimit() != nil && q.GetLimit().GetValue() >= 0 && start+int(q.GetLimit().GetValue()) < end {
		end = start + int(q.GetLimit().GetValue())
		limited = true
	}
	batch := &dspb.QueryResultBatch{
		SkippedResults:   int32(q.GetOffset()),
		EntityResultType: dspb.EntityResult_FULL,
		ReadTime:         timestamppb.Now(),
	}
	if len(q.GetProjection()) > 0 {
		batch.EntityResultType = dspb.EntityResult_PROJECTION
	}
	for _, rec := range records[start:end] {
		batch.EntityResults = append(batch.EntityResults, resultFor(rec, batch.EntityResultType, q.GetProjection()))
	}
	if limited || end < len(records) {
		batch.EndCursor = []byte(strconv.Itoa(end))
		if limited {
			batch.MoreResults = dspb.QueryResultBatch_MORE_RESULTS_AFTER_LIMIT
		} else {
			batch.MoreResults = dspb.QueryResultBatch_MORE_RESULTS_AFTER_CURSOR
		}
	} else {
		batch.EndCursor = []byte(strconv.Itoa(end))
		batch.MoreResults = dspb.QueryResultBatch_NO_MORE_RESULTS
	}
	return batch, nil
}

func (s *Server) RunQuery(ctx context.Context, req *dspb.RunQueryRequest) (*dspb.RunQueryResponse, error) {
	if req.GetProjectId() == "" {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "project_id is required")
	}
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	data, err := s.state.dataForRead(req.GetReadOptions().GetTransaction())
	if err != nil {
		return nil, err
	}
	batch, err := s.runQuery(data, req)
	if err != nil {
		return nil, err
	}
	resp := &dspb.RunQueryResponse{Batch: batch}
	if req.GetGqlQuery() != nil {
		resp.Query = req.GetQuery()
	}
	if req.GetReadOptions().GetNewTransaction() != nil {
		tx, err := s.beginTransactionLocked()
		if err != nil {
			return nil, err
		}
		resp.Transaction = tx
	}
	return resp, nil
}

func (s *Server) beginTransactionLocked() ([]byte, error) {
	id := newTransactionID()
	s.state.transactions[string(id)] = &transaction{data: cloneData(s.state.entities), active: true}
	return id, nil
}

func (s *Server) BeginTransaction(ctx context.Context, req *dspb.BeginTransactionRequest) (*dspb.BeginTransactionResponse, error) {
	if req.GetProjectId() == "" {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "project_id is required")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	id, err := s.beginTransactionLocked()
	if err != nil {
		return nil, err
	}
	return &dspb.BeginTransactionResponse{Transaction: id}, nil
}

func keyIsIncomplete(key *dspb.Key) bool {
	if key == nil || len(key.Path) == 0 {
		return true
	}
	last := key.Path[len(key.Path)-1]
	return last.GetName() == "" && last.GetId() == 0
}

func completeKey(key *dspb.Key, id int64) (*dspb.Key, bool) {
	out := proto.Clone(key).(*dspb.Key)
	if !keyIsIncomplete(out) {
		return out, false
	}
	out.Path[len(out.Path)-1].IdType = &dspb.Key_PathElement_Id{Id: id}
	return out, true
}

func applyMutation(data map[string]map[string]*entityRecord, mut *dspb.Mutation, project, database string, nextID func() int64, version int64) (*dspb.MutationResult, error) {
	now := timestamppb.Now()
	result := &dspb.MutationResult{Version: version}
	var ent *dspb.Entity
	var key *dspb.Key
	op := "delete"
	switch {
	case mut.GetInsert() != nil:
		ent, key, op = cloneEntity(mut.GetInsert()), mut.GetInsert().GetKey(), "insert"
	case mut.GetUpdate() != nil:
		ent, key, op = cloneEntity(mut.GetUpdate()), mut.GetUpdate().GetKey(), "update"
	case mut.GetUpsert() != nil:
		ent, key, op = cloneEntity(mut.GetUpsert()), mut.GetUpsert().GetKey(), "upsert"
	case mut.GetDelete() != nil:
		key = mut.GetDelete()
	default:
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "mutation operation is required")
	}
	key = normalizeKey(key, project, database, "")
	allocated := false
	if ent != nil {
		key, allocated = completeKey(key, nextID())
		ent.Key = key
		if allocated {
			result.Key = proto.Clone(key).(*dspb.Key)
		}
	}
	scope, canonical, ok := canonicalKey(key)
	if !ok {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "mutation key must have a non-empty path")
	}
	path := strings.TrimPrefix(canonical, scope+"\x00")
	if data[scope] == nil {
		data[scope] = map[string]*entityRecord{}
	}
	existing, exists := data[scope][path]
	if mut.GetBaseVersion() != 0 {
		current := int64(0)
		if exists {
			current = existing.version
		}
		if current != mut.GetBaseVersion() {
			result.ConflictDetected = true
			if exists {
				result.Version = existing.version
				result.CreateTime = cloneTime(existing.createTime)
				result.UpdateTime = cloneTime(existing.updateTime)
			}
			return result, nil
		}
	}
	switch op {
	case "insert":
		if exists {
			return nil, gcperr.GRPC(gcperr.AlreadyExists, "entity already exists")
		}
		data[scope][path] = &entityRecord{entity: ent, version: version, createTime: now, updateTime: now}
		result.CreateTime, result.UpdateTime = cloneTime(now), cloneTime(now)
	case "update":
		if !exists {
			return nil, gcperr.GRPC(gcperr.NotFound, "entity not found")
		}
		if mask := mut.GetPropertyMask(); mask != nil && len(mask.GetPaths()) > 0 {
			merged := cloneEntity(existing.entity)
			for _, path := range mask.GetPaths() {
				if v, ok := ent.Properties[path]; ok {
					merged.Properties[path] = proto.Clone(v).(*dspb.Value)
				} else {
					delete(merged.Properties, path)
				}
			}
			ent = merged
		}
		data[scope][path] = &entityRecord{entity: ent, version: version, createTime: cloneTime(existing.createTime), updateTime: now}
		result.CreateTime, result.UpdateTime = cloneTime(existing.createTime), cloneTime(now)
	case "upsert":
		created := cloneTime(now)
		if exists {
			created = cloneTime(existing.createTime)
		}
		data[scope][path] = &entityRecord{entity: ent, version: version, createTime: created, updateTime: now}
		result.CreateTime, result.UpdateTime = cloneTime(created), cloneTime(now)
	case "delete":
		delete(data[scope], path)
		if exists {
			result.Version = version
		}
	}
	return result, nil
}

func (s *Server) Commit(ctx context.Context, req *dspb.CommitRequest) (*dspb.CommitResponse, error) {
	if req.GetProjectId() == "" {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "project_id is required")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	data := s.state.entities
	transactional := req.GetMode() != dspb.CommitRequest_NON_TRANSACTIONAL
	if transactional {
		tx, ok := s.state.transactions[string(req.GetTransaction())]
		if !ok || !tx.active {
			return nil, gcperr.GRPC(gcperr.InvalidArgument, "transaction is not active")
		}
		data = tx.data
	}
	// Validate and apply onto the selected data snapshot. Transactional commits
	// publish the whole snapshot only after every mutation succeeds.
	results := make([]*dspb.MutationResult, 0, len(req.GetMutations()))
	for _, mut := range req.GetMutations() {
		version := s.state.nextVersion
		s.state.nextVersion++
		nextID := func() int64 { id := s.state.nextID; s.state.nextID += 1 + int64(time.Now().UnixNano()%97); return id }
		res, err := applyMutation(data, mut, req.GetProjectId(), req.GetDatabaseId(), nextID, version)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	if transactional {
		s.state.entities = data
		delete(s.state.transactions, string(req.GetTransaction()))
	}
	return &dspb.CommitResponse{MutationResults: results, IndexUpdates: int32(len(results)), CommitTime: timestamppb.Now()}, nil
}

func (s *Server) Rollback(ctx context.Context, req *dspb.RollbackRequest) (*dspb.RollbackResponse, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	if _, ok := s.state.transactions[string(req.GetTransaction())]; !ok {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "transaction is not active")
	}
	delete(s.state.transactions, string(req.GetTransaction()))
	return &dspb.RollbackResponse{}, nil
}

func (s *Server) AllocateIds(ctx context.Context, req *dspb.AllocateIdsRequest) (*dspb.AllocateIdsResponse, error) {
	if len(req.GetKeys()) == 0 {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "keys are required")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	out := make([]*dspb.Key, 0, len(req.GetKeys()))
	for _, input := range req.GetKeys() {
		key := normalizeKey(input, req.GetProjectId(), req.GetDatabaseId(), "")
		if !keyIsIncomplete(key) {
			return nil, gcperr.GRPC(gcperr.InvalidArgument, "AllocateIds keys must be incomplete")
		}
		allocated, _ := completeKey(key, s.state.nextID)
		s.state.nextID += 1 + int64(time.Now().UnixNano()%97)
		out = append(out, allocated)
	}
	return &dspb.AllocateIdsResponse{Keys: out}, nil
}

func (s *Server) ReserveIds(ctx context.Context, req *dspb.ReserveIdsRequest) (*dspb.ReserveIdsResponse, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	for _, key := range req.GetKeys() {
		if len(key.GetPath()) > 0 {
			if id := key.GetPath()[len(key.GetPath())-1].GetId(); id >= s.state.nextID {
				s.state.nextID = id + 1
			}
		}
	}
	return &dspb.ReserveIdsResponse{}, nil
}

// Keep fmt referenced in this file's API surface for useful nested errors in
// future validation paths.
var _ = fmt.Sprintf

func (s *Server) RunAggregationQuery(ctx context.Context, req *dspb.RunAggregationQueryRequest) (*dspb.RunAggregationQueryResponse, error) {
	aq := req.GetAggregationQuery()
	if aq == nil || aq.GetNestedQuery() == nil {
		return nil, gcperr.GRPC(gcperr.Unimplemented, "GQL aggregation queries are not implemented; use structured aggregation queries")
	}
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	data, err := s.state.dataForRead(req.GetReadOptions().GetTransaction())
	if err != nil {
		return nil, err
	}
	nestedReq := &dspb.RunQueryRequest{
		ProjectId: req.GetProjectId(), DatabaseId: req.GetDatabaseId(), PartitionId: req.GetPartitionId(),
		QueryType: &dspb.RunQueryRequest_Query{Query: aq.GetNestedQuery()},
	}
	batch, err := s.runQuery(data, nestedReq)
	if err != nil {
		return nil, err
	}
	props := map[string]*dspb.Value{}
	for i, aggregation := range aq.GetAggregations() {
		alias := aggregation.GetAlias()
		if alias == "" {
			alias = fmt.Sprintf("property_%d", i+1)
		}
		switch {
		case aggregation.GetCount() != nil:
			count := int64(len(batch.GetEntityResults()))
			if upTo := aggregation.GetCount().GetUpTo(); upTo != nil && count > upTo.GetValue() {
				count = upTo.GetValue()
			}
			props[alias] = &dspb.Value{ValueType: &dspb.Value_IntegerValue{IntegerValue: count}}
		case aggregation.GetSum() != nil || aggregation.GetAvg() != nil:
			var property string
			if aggregation.GetSum() != nil {
				property = aggregation.GetSum().GetProperty().GetName()
			} else {
				property = aggregation.GetAvg().GetProperty().GetName()
			}
			var sum float64
			var n int64
			for _, result := range batch.GetEntityResults() {
				if v, ok := valueForProperty(result.GetEntity(), property); ok {
					sum += numericValue(v)
					n++
				}
			}
			value := sum
			if aggregation.GetAvg() != nil {
				if n == 0 {
					value = 0
				} else {
					value = sum / float64(n)
				}
			}
			props[alias] = &dspb.Value{ValueType: &dspb.Value_DoubleValue{DoubleValue: value}}
		default:
			return nil, gcperr.GRPC(gcperr.InvalidArgument, "aggregation operator is required")
		}
	}
	return &dspb.RunAggregationQueryResponse{Batch: &dspb.AggregationResultBatch{
		AggregationResults: []*dspb.AggregationResult{{AggregateProperties: props}},
		MoreResults:        dspb.QueryResultBatch_NO_MORE_RESULTS,
		ReadTime:           timestamppb.Now(),
	}, Query: aq}, nil
}
