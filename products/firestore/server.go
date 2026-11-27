// Package firestore emulates the google.firestore.v1 document data plane in memory.
package firestore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	fspb "github.com/bestony/google-cloud-emulator/gen/google/firestore/v1"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type documentRecord struct {
	document *fspb.Document
}

type transaction struct {
	data   map[string]map[string]*documentRecord
	active bool
}

// State stores documents by database resource name and database-relative path.
type State struct {
	mu           sync.RWMutex
	documents    map[string]map[string]*documentRecord
	transactions map[string]*transaction
}

func NewState() *State {
	return &State{documents: map[string]map[string]*documentRecord{}, transactions: map[string]*transaction{}}
}

// Server implements FirestoreServer.
type Server struct {
	fspb.UnimplementedFirestoreServer
	state *State
}

func NewServer(st *State) *Server {
	if st == nil {
		st = NewState()
	}
	return &Server{state: st}
}

func validDatabase(name string) bool {
	parts := strings.Split(name, "/")
	return len(parts) == 4 && parts[0] == "projects" && parts[1] != "" && parts[2] == "databases" && parts[3] != ""
}

func splitDocumentName(name string) (database, rel string, ok bool) {
	idx := strings.Index(name, "/documents/")
	if idx < 0 {
		return "", "", false
	}
	database = name[:idx]
	rel = strings.Trim(strings.TrimPrefix(name[idx:], "/documents/"), "/")
	return database, rel, validDatabase(database) && rel != ""
}

func documentName(database, rel string) string {
	return strings.TrimSuffix(database, "/") + "/documents/" + strings.Trim(rel, "/")
}

func parentRelative(parent string) string {
	idx := strings.Index(parent, "/documents")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimPrefix(parent[idx:], "/documents")
	return strings.Trim(rest, "/")
}

func cloneDocument(in *fspb.Document) *fspb.Document {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*fspb.Document)
}
func cloneRecord(in *documentRecord) *documentRecord {
	return &documentRecord{document: cloneDocument(in.document)}
}
func cloneData(in map[string]map[string]*documentRecord) map[string]map[string]*documentRecord {
	out := make(map[string]map[string]*documentRecord, len(in))
	for db, docs := range in {
		out[db] = make(map[string]*documentRecord, len(docs))
		for path, rec := range docs {
			out[db][path] = cloneRecord(rec)
		}
	}
	return out
}

func transactionID() []byte {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return []byte(hex.EncodeToString(b))
}

func randomDocumentID() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	out := make([]byte, 20)
	for i := range out {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out)
}

func (s *State) dataFor(tx []byte) (map[string]map[string]*documentRecord, error) {
	if len(tx) == 0 {
		return s.documents, nil
	}
	transaction, ok := s.transactions[string(tx)]
	if !ok || !transaction.active {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "transaction is not active")
	}
	return transaction.data, nil
}

func fieldValue(fields map[string]*fspb.Value, path string) (*fspb.Value, bool) {
	parts := strings.Split(path, ".")
	var value *fspb.Value
	for i, part := range parts {
		v, ok := fields[part]
		if !ok {
			return nil, false
		}
		if i == len(parts)-1 {
			value = v
			break
		}
		if v.GetMapValue() == nil {
			return nil, false
		}
		fields = v.GetMapValue().GetFields()
	}
	return value, value != nil
}

func setField(fields map[string]*fspb.Value, path string, value *fspb.Value) {
	parts := strings.Split(path, ".")
	for _, part := range parts[:len(parts)-1] {
		child, ok := fields[part]
		if !ok || child.GetMapValue() == nil {
			child = &fspb.Value{ValueType: &fspb.Value_MapValue{MapValue: &fspb.MapValue{Fields: map[string]*fspb.Value{}}}}
			fields[part] = child
		}
		fields = child.GetMapValue().Fields
	}
	fields[parts[len(parts)-1]] = proto.Clone(value).(*fspb.Value)
}

func deleteField(fields map[string]*fspb.Value, path string) {
	parts := strings.Split(path, ".")
	for _, part := range parts[:len(parts)-1] {
		child, ok := fields[part]
		if !ok || child.GetMapValue() == nil {
			return
		}
		fields = child.GetMapValue().GetFields()
	}
	delete(fields, parts[len(parts)-1])
}

func valueRank(v *fspb.Value) int {
	switch v.GetValueType().(type) {
	case *fspb.Value_NullValue:
		return 0
	case *fspb.Value_BooleanValue:
		return 1
	case *fspb.Value_IntegerValue, *fspb.Value_DoubleValue:
		return 2
	case *fspb.Value_TimestampValue:
		return 3
	case *fspb.Value_StringValue:
		return 4
	case *fspb.Value_BytesValue:
		return 5
	case *fspb.Value_ReferenceValue:
		return 6
	case *fspb.Value_GeoPointValue:
		return 7
	case *fspb.Value_ArrayValue:
		return 8
	case *fspb.Value_MapValue:
		return 9
	default:
		return 10
	}
}

func numeric(v *fspb.Value) float64 {
	if i, ok := v.GetValueType().(*fspb.Value_IntegerValue); ok {
		return float64(i.IntegerValue)
	}
	return v.GetDoubleValue()
}

func compareValues(a, b *fspb.Value) int {
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
		av, bv := numeric(a), numeric(b)
		if math.IsNaN(av) && math.IsNaN(bv) {
			return 0
		}
		if math.IsNaN(av) {
			return -1
		}
		if math.IsNaN(bv) {
			return 1
		}
		if av == bv {
			return 0
		}
		if av < bv {
			return -1
		}
		return 1
	case 3:
		av, bv := a.GetTimestampValue().AsTime(), b.GetTimestampValue().AsTime()
		if av.Equal(bv) {
			return 0
		}
		if av.Before(bv) {
			return -1
		}
		return 1
	case 4:
		return strings.Compare(a.GetStringValue(), b.GetStringValue())
	case 5:
		return strings.Compare(string(a.GetBytesValue()), string(b.GetBytesValue()))
	case 6:
		return strings.Compare(a.GetReferenceValue(), b.GetReferenceValue())
	default:
		if proto.Equal(a, b) {
			return 0
		}
		return strings.Compare(a.String(), b.String())
	}
}
func valuesEqual(a, b *fspb.Value) bool { return compareValues(a, b) == 0 }

func maskDocument(doc *fspb.Document, mask *fspb.DocumentMask) *fspb.Document {
	out := cloneDocument(doc)
	if mask == nil || len(mask.GetFieldPaths()) == 0 {
		return out
	}
	fields := map[string]*fspb.Value{}
	for _, path := range mask.GetFieldPaths() {
		if v, ok := fieldValue(doc.GetFields(), path); ok {
			setField(fields, path, v)
		}
	}
	out.Fields = fields
	return out
}

func checkPrecondition(pre *fspb.Precondition, rec *documentRecord) error {
	if pre == nil {
		return nil
	}
	exists := rec != nil
	if _, hasExists := pre.GetConditionType().(*fspb.Precondition_Exists); hasExists {
		if exists != pre.GetExists() {
			if pre.GetExists() {
				return gcperr.GRPC(gcperr.FailedPrecondition, "document does not exist")
			}
			return gcperr.GRPC(gcperr.FailedPrecondition, "document already exists")
		}
	}
	if pre.GetUpdateTime() != nil {
		if rec == nil || !rec.document.GetUpdateTime().AsTime().Equal(pre.GetUpdateTime().AsTime()) {
			return gcperr.GRPC(gcperr.FailedPrecondition, "document update time precondition failed")
		}
	}
	return nil
}

func upsertDocument(data map[string]map[string]*documentRecord, doc *fspb.Document, mask *fspb.DocumentMask, pre *fspb.Precondition, now *timestamppb.Timestamp) (*fspb.Document, error) {
	db, rel, ok := splitDocumentName(doc.GetName())
	if !ok {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid document name")
	}
	if data[db] == nil {
		data[db] = map[string]*documentRecord{}
	}
	existing := data[db][rel]
	if err := checkPrecondition(pre, existing); err != nil {
		return nil, err
	}
	out := cloneDocument(doc)
	out.Name = documentName(db, rel)
	create := now
	if existing != nil {
		create = existing.document.GetCreateTime()
		if mask != nil && len(mask.GetFieldPaths()) > 0 {
			merged := cloneDocument(existing.document)
			if merged.Fields == nil {
				merged.Fields = map[string]*fspb.Value{}
			}
			for _, path := range mask.GetFieldPaths() {
				if v, ok := fieldValue(doc.GetFields(), path); ok {
					setField(merged.Fields, path, v)
				} else {
					deleteField(merged.Fields, path)
				}
			}
			merged.Name = out.Name
			out = merged
		}
	}
	out.CreateTime = create
	out.UpdateTime = now
	if out.Fields == nil {
		out.Fields = map[string]*fspb.Value{}
	}
	data[db][rel] = &documentRecord{document: cloneDocument(out)}
	return cloneDocument(out), nil
}

func arrayValue(values ...*fspb.Value) *fspb.Value {
	cp := make([]*fspb.Value, 0, len(values))
	for _, v := range values {
		cp = append(cp, proto.Clone(v).(*fspb.Value))
	}
	return &fspb.Value{ValueType: &fspb.Value_ArrayValue{ArrayValue: &fspb.ArrayValue{Values: cp}}}
}

func applyTransforms(doc *fspb.Document, transforms []*fspb.DocumentTransform_FieldTransform, now *timestamppb.Timestamp) []*fspb.Value {
	if doc.Fields == nil {
		doc.Fields = map[string]*fspb.Value{}
	}
	results := make([]*fspb.Value, 0, len(transforms))
	for _, tr := range transforms {
		path := tr.GetFieldPath()
		cur, exists := fieldValue(doc.Fields, path)
		var out *fspb.Value
		switch {
		case tr.GetSetToServerValue() == fspb.DocumentTransform_FieldTransform_REQUEST_TIME:
			out = &fspb.Value{ValueType: &fspb.Value_TimestampValue{TimestampValue: now}}
		case tr.GetIncrement() != nil:
			delta := tr.GetIncrement()
			if !exists {
				cur = &fspb.Value{ValueType: &fspb.Value_IntegerValue{IntegerValue: 0}}
			}
			if _, ai := cur.GetValueType().(*fspb.Value_IntegerValue); ai {
				if _, bi := delta.GetValueType().(*fspb.Value_IntegerValue); bi {
					out = &fspb.Value{ValueType: &fspb.Value_IntegerValue{IntegerValue: cur.GetIntegerValue() + delta.GetIntegerValue()}}
				} else {
					out = &fspb.Value{ValueType: &fspb.Value_DoubleValue{DoubleValue: numeric(cur) + numeric(delta)}}
				}
			} else {
				out = &fspb.Value{ValueType: &fspb.Value_DoubleValue{DoubleValue: numeric(cur) + numeric(delta)}}
			}
		case tr.GetMaximum() != nil:
			if !exists || compareValues(cur, tr.GetMaximum()) < 0 {
				out = proto.Clone(tr.GetMaximum()).(*fspb.Value)
			} else {
				out = proto.Clone(cur).(*fspb.Value)
			}
		case tr.GetMinimum() != nil:
			if !exists || compareValues(cur, tr.GetMinimum()) > 0 {
				out = proto.Clone(tr.GetMinimum()).(*fspb.Value)
			} else {
				out = proto.Clone(cur).(*fspb.Value)
			}
		case tr.GetAppendMissingElements() != nil:
			vals := []*fspb.Value{}
			if exists && cur.GetArrayValue() != nil {
				vals = append(vals, cur.GetArrayValue().GetValues()...)
			}
			for _, candidate := range tr.GetAppendMissingElements().GetValues() {
				found := false
				for _, have := range vals {
					if valuesEqual(have, candidate) {
						found = true
						break
					}
				}
				if !found {
					vals = append(vals, candidate)
				}
			}
			out = arrayValue(vals...)
		case tr.GetRemoveAllFromArray() != nil:
			vals := []*fspb.Value{}
			if exists && cur.GetArrayValue() != nil {
				for _, have := range cur.GetArrayValue().GetValues() {
					remove := false
					for _, candidate := range tr.GetRemoveAllFromArray().GetValues() {
						if valuesEqual(have, candidate) {
							remove = true
							break
						}
					}
					if !remove {
						vals = append(vals, have)
					}
				}
			}
			out = arrayValue(vals...)
		default:
			continue
		}
		setField(doc.Fields, path, out)
		results = append(results, proto.Clone(out).(*fspb.Value))
	}
	return results
}

func applyWrite(data map[string]map[string]*documentRecord, write *fspb.Write, now *timestamppb.Timestamp) (*fspb.WriteResult, error) {
	result := &fspb.WriteResult{UpdateTime: now}
	if write.GetUpdate() != nil {
		doc, err := upsertDocument(data, write.GetUpdate(), write.GetUpdateMask(), write.GetCurrentDocument(), now)
		if err != nil {
			return nil, err
		}
		if len(write.GetUpdateTransforms()) > 0 {
			mutable := cloneDocument(doc)
			result.TransformResults = applyTransforms(mutable, write.GetUpdateTransforms(), now)
			if _, err := upsertDocument(data, mutable, nil, nil, now); err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	if name := write.GetDelete(); name != "" {
		db, rel, ok := splitDocumentName(name)
		if !ok {
			return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid document name")
		}
		existing := data[db][rel]
		if err := checkPrecondition(write.GetCurrentDocument(), existing); err != nil {
			return nil, err
		}
		delete(data[db], rel)
		return result, nil
	}
	if tr := write.GetTransform(); tr != nil {
		db, rel, ok := splitDocumentName(tr.GetDocument())
		if !ok {
			return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid document name")
		}
		if data[db] == nil {
			data[db] = map[string]*documentRecord{}
		}
		rec := data[db][rel]
		if err := checkPrecondition(write.GetCurrentDocument(), rec); err != nil {
			return nil, err
		}
		doc := &fspb.Document{Name: documentName(db, rel), Fields: map[string]*fspb.Value{}, CreateTime: now}
		if rec != nil {
			doc = cloneDocument(rec.document)
		}
		result.TransformResults = applyTransforms(doc, tr.GetFieldTransforms(), now)
		doc.UpdateTime = now
		if doc.CreateTime == nil {
			doc.CreateTime = now
		}
		data[db][rel] = &documentRecord{document: cloneDocument(doc)}
		return result, nil
	}
	return nil, gcperr.GRPC(gcperr.InvalidArgument, "write operation is required")
}

func (s *Server) GetDocument(ctx context.Context, req *fspb.GetDocumentRequest) (*fspb.Document, error) {
	db, rel, ok := splitDocumentName(req.GetName())
	if !ok {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid document name")
	}
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	data, err := s.state.dataFor(req.GetTransaction())
	if err != nil {
		return nil, err
	}
	rec := data[db][rel]
	if rec == nil {
		return nil, gcperr.GRPC(gcperr.NotFound, "document not found")
	}
	return maskDocument(rec.document, req.GetMask()), nil
}

func (s *Server) CreateDocument(ctx context.Context, req *fspb.CreateDocumentRequest) (*fspb.Document, error) {
	// parent may itself be the database documents root; derive its relative path manually.
	idx := strings.Index(req.GetParent(), "/documents")
	if idx < 0 {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid parent")
	}
	database := req.GetParent()[:idx]
	if !validDatabase(database) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid parent database")
	}
	relParent := parentRelative(req.GetParent())
	if req.GetCollectionId() == "" {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "collection_id is required")
	}
	id := req.GetDocumentId()
	if id == "" {
		id = randomDocumentID()
	}
	rel := strings.Trim(strings.Join([]string{relParent, req.GetCollectionId(), id}, "/"), "/")
	doc := cloneDocument(req.GetDocument())
	if doc == nil {
		doc = &fspb.Document{}
	}
	doc.Name = documentName(database, rel)
	now := timestamppb.Now()
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	if s.state.documents[database] == nil {
		s.state.documents[database] = map[string]*documentRecord{}
	}
	if s.state.documents[database][rel] != nil {
		return nil, gcperr.GRPC(gcperr.AlreadyExists, "document already exists")
	}
	return upsertDocument(s.state.documents, doc, nil, nil, now)
}

func (s *Server) UpdateDocument(ctx context.Context, req *fspb.UpdateDocumentRequest) (*fspb.Document, error) {
	if req.GetDocument() == nil {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "document is required")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return upsertDocument(s.state.documents, req.GetDocument(), req.GetUpdateMask(), req.GetCurrentDocument(), timestamppb.Now())
}

func (s *Server) DeleteDocument(ctx context.Context, req *fspb.DeleteDocumentRequest) (*emptypb.Empty, error) {
	db, rel, ok := splitDocumentName(req.GetName())
	if !ok {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid document name")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	rec := s.state.documents[db][rel]
	if err := checkPrecondition(req.GetCurrentDocument(), rec); err != nil {
		return nil, err
	}
	delete(s.state.documents[db], rel)
	return &emptypb.Empty{}, nil
}

func (s *Server) ListDocuments(ctx context.Context, req *fspb.ListDocumentsRequest) (*fspb.ListDocumentsResponse, error) {
	idx := strings.Index(req.GetParent(), "/documents")
	if idx < 0 {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid parent")
	}
	database := req.GetParent()[:idx]
	prefix := strings.Trim(strings.Join([]string{parentRelative(req.GetParent()), req.GetCollectionId()}, "/"), "/")
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	data, err := s.state.dataFor(req.GetTransaction())
	if err != nil {
		return nil, err
	}
	var names []string
	for path := range data[database] {
		if !strings.HasPrefix(path, prefix+"/") {
			continue
		}
		rest := strings.TrimPrefix(path, prefix+"/")
		if !strings.Contains(rest, "/") {
			names = append(names, path)
		}
	}
	sort.Strings(names)
	start, _ := strconv.Atoi(req.GetPageToken())
	if start > len(names) {
		start = len(names)
	}
	size := int(req.GetPageSize())
	if size <= 0 || size > 300 {
		size = 300
	}
	end := start + size
	if end > len(names) {
		end = len(names)
	}
	resp := &fspb.ListDocumentsResponse{}
	for _, path := range names[start:end] {
		resp.Documents = append(resp.Documents, maskDocument(data[database][path].document, req.GetMask()))
	}
	if end < len(names) {
		resp.NextPageToken = strconv.Itoa(end)
	}
	return resp, nil
}

func (s *Server) BatchGetDocuments(req *fspb.BatchGetDocumentsRequest, stream grpc.ServerStreamingServer[fspb.BatchGetDocumentsResponse]) error {
	if !validDatabase(req.GetDatabase()) {
		return gcperr.GRPC(gcperr.InvalidArgument, "invalid database")
	}
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	data, err := s.state.dataFor(req.GetTransaction())
	if err != nil {
		return err
	}
	for _, name := range req.GetDocuments() {
		db, rel, ok := splitDocumentName(name)
		if !ok || db != req.GetDatabase() {
			return gcperr.GRPC(gcperr.InvalidArgument, "document %q is outside database", name)
		}
		resp := &fspb.BatchGetDocumentsResponse{ReadTime: timestamppb.Now()}
		if rec := data[db][rel]; rec != nil {
			resp.Result = &fspb.BatchGetDocumentsResponse_Found{Found: maskDocument(rec.document, req.GetMask())}
		} else {
			resp.Result = &fspb.BatchGetDocumentsResponse_Missing{Missing: name}
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func matchesFieldFilter(doc *fspb.Document, ff *fspb.StructuredQuery_FieldFilter) bool {
	path := ff.GetField().GetFieldPath()
	actual, exists := fieldValue(doc.GetFields(), path)
	if path == "__name__" {
		actual = &fspb.Value{ValueType: &fspb.Value_ReferenceValue{ReferenceValue: doc.GetName()}}
		exists = true
	}
	if !exists {
		return false
	}
	want := ff.GetValue()
	cmp := compareValues(actual, want)
	switch ff.GetOp() {
	case fspb.StructuredQuery_FieldFilter_LESS_THAN:
		return cmp < 0
	case fspb.StructuredQuery_FieldFilter_LESS_THAN_OR_EQUAL:
		return cmp <= 0
	case fspb.StructuredQuery_FieldFilter_GREATER_THAN:
		return cmp > 0
	case fspb.StructuredQuery_FieldFilter_GREATER_THAN_OR_EQUAL:
		return cmp >= 0
	case fspb.StructuredQuery_FieldFilter_EQUAL:
		return cmp == 0
	case fspb.StructuredQuery_FieldFilter_NOT_EQUAL:
		return cmp != 0
	case fspb.StructuredQuery_FieldFilter_ARRAY_CONTAINS:
		for _, v := range actual.GetArrayValue().GetValues() {
			if valuesEqual(v, want) {
				return true
			}
		}
	case fspb.StructuredQuery_FieldFilter_IN:
		for _, v := range want.GetArrayValue().GetValues() {
			if valuesEqual(actual, v) {
				return true
			}
		}
	case fspb.StructuredQuery_FieldFilter_ARRAY_CONTAINS_ANY:
		for _, have := range actual.GetArrayValue().GetValues() {
			for _, candidate := range want.GetArrayValue().GetValues() {
				if valuesEqual(have, candidate) {
					return true
				}
			}
		}
	case fspb.StructuredQuery_FieldFilter_NOT_IN:
		for _, v := range want.GetArrayValue().GetValues() {
			if valuesEqual(actual, v) {
				return false
			}
		}
		return true
	}
	return false
}

func matchesFilter(doc *fspb.Document, filter *fspb.StructuredQuery_Filter) bool {
	if filter == nil {
		return true
	}
	if ff := filter.GetFieldFilter(); ff != nil {
		return matchesFieldFilter(doc, ff)
	}
	if uf := filter.GetUnaryFilter(); uf != nil {
		v, exists := fieldValue(doc.GetFields(), uf.GetField().GetFieldPath())
		if !exists {
			return false
		}
		_, isNull := v.GetValueType().(*fspb.Value_NullValue)
		isNaN := false
		if _, numericType := v.GetValueType().(*fspb.Value_DoubleValue); numericType {
			isNaN = math.IsNaN(v.GetDoubleValue())
		}
		switch uf.GetOp() {
		case fspb.StructuredQuery_UnaryFilter_IS_NULL:
			return isNull
		case fspb.StructuredQuery_UnaryFilter_IS_NOT_NULL:
			return !isNull
		case fspb.StructuredQuery_UnaryFilter_IS_NAN:
			return isNaN
		case fspb.StructuredQuery_UnaryFilter_IS_NOT_NAN:
			return !isNaN
		}
		return false
	}
	cf := filter.GetCompositeFilter()
	if cf == nil {
		return true
	}
	if cf.GetOp() == fspb.StructuredQuery_CompositeFilter_OR {
		for _, child := range cf.GetFilters() {
			if matchesFilter(doc, child) {
				return true
			}
		}
		return false
	}
	for _, child := range cf.GetFilters() {
		if !matchesFilter(doc, child) {
			return false
		}
	}
	return true
}

func documentInQuery(parentRel string, selector *fspb.StructuredQuery_CollectionSelector, path string) bool {
	parts := strings.Split(path, "/")
	if selector.GetAllDescendants() {
		return len(parts) >= 2 && parts[len(parts)-2] == selector.GetCollectionId()
	}
	collectionPath := strings.Trim(strings.Join([]string{parentRel, selector.GetCollectionId()}, "/"), "/")
	if !strings.HasPrefix(path, collectionPath+"/") {
		return false
	}
	rest := strings.TrimPrefix(path, collectionPath+"/")
	return !strings.Contains(rest, "/")
}

func applyProjection(doc *fspb.Document, q *fspb.StructuredQuery) *fspb.Document {
	if q.GetSelect() == nil || len(q.GetSelect().GetFields()) == 0 {
		return cloneDocument(doc)
	}
	out := &fspb.Document{Name: doc.GetName(), CreateTime: doc.GetCreateTime(), UpdateTime: doc.GetUpdateTime(), Fields: map[string]*fspb.Value{}}
	for _, field := range q.GetSelect().GetFields() {
		if v, ok := fieldValue(doc.GetFields(), field.GetFieldPath()); ok {
			setField(out.Fields, field.GetFieldPath(), v)
		}
	}
	return out
}

func (s *Server) queryDocuments(data map[string]map[string]*documentRecord, parent string, q *fspb.StructuredQuery) ([]*fspb.Document, error) {
	idx := strings.Index(parent, "/documents")
	if idx < 0 {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid parent")
	}
	database := parent[:idx]
	parentRel := parentRelative(parent)
	if len(q.GetFrom()) != 1 {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "exactly one collection selector is required")
	}
	selector := q.GetFrom()[0]
	var docs []*fspb.Document
	for path, rec := range data[database] {
		if documentInQuery(parentRel, selector, path) && matchesFilter(rec.document, q.GetWhere()) {
			docs = append(docs, cloneDocument(rec.document))
		}
	}
	for i := len(q.GetOrderBy()) - 1; i >= 0; i-- {
		order := q.GetOrderBy()[i]
		sort.SliceStable(docs, func(a, b int) bool {
			av, aok := fieldValue(docs[a].GetFields(), order.GetField().GetFieldPath())
			bv, bok := fieldValue(docs[b].GetFields(), order.GetField().GetFieldPath())
			if order.GetField().GetFieldPath() == "__name__" {
				av = &fspb.Value{ValueType: &fspb.Value_ReferenceValue{ReferenceValue: docs[a].GetName()}}
				bv = &fspb.Value{ValueType: &fspb.Value_ReferenceValue{ReferenceValue: docs[b].GetName()}}
				aok = true
				bok = true
			}
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
			if order.GetDirection() == fspb.StructuredQuery_DESCENDING {
				return cmp > 0
			}
			return cmp < 0
		})
	}
	offset := int(q.GetOffset())
	if offset > len(docs) {
		offset = len(docs)
	}
	docs = docs[offset:]
	if q.GetLimit() != nil && q.GetLimit().GetValue() >= 0 && int(q.GetLimit().GetValue()) < len(docs) {
		docs = docs[:q.GetLimit().GetValue()]
	}
	for i, doc := range docs {
		docs[i] = applyProjection(doc, q)
	}
	return docs, nil
}

func (s *Server) RunQuery(req *fspb.RunQueryRequest, stream grpc.ServerStreamingServer[fspb.RunQueryResponse]) error {
	q := req.GetStructuredQuery()
	if q == nil {
		return gcperr.GRPC(gcperr.InvalidArgument, "structured_query is required")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	data, err := s.state.dataFor(req.GetTransaction())
	if err != nil {
		return err
	}
	if req.GetNewTransaction() != nil {
		id := transactionID()
		s.state.transactions[string(id)] = &transaction{data: cloneData(s.state.documents), active: true}
		if err := stream.Send(&fspb.RunQueryResponse{Transaction: id}); err != nil {
			return err
		}
	}
	docs, err := s.queryDocuments(data, req.GetParent(), q)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return stream.Send(&fspb.RunQueryResponse{ReadTime: timestamppb.Now(), ContinuationSelector: &fspb.RunQueryResponse_Done{Done: true}})
	}
	for i, doc := range docs {
		resp := &fspb.RunQueryResponse{Document: doc, ReadTime: timestamppb.Now()}
		if i == 0 {
			resp.SkippedResults = q.GetOffset()
		}
		if i == len(docs)-1 {
			resp.ContinuationSelector = &fspb.RunQueryResponse_Done{Done: true}
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func aggregationNumber(v *fspb.Value) (float64, bool, bool) {
	if v == nil {
		return 0, false, false
	}
	if i, ok := v.GetValueType().(*fspb.Value_IntegerValue); ok {
		return float64(i.IntegerValue), true, true
	}
	if d, ok := v.GetValueType().(*fspb.Value_DoubleValue); ok {
		return d.DoubleValue, true, false
	}
	return 0, false, false
}

func (s *Server) RunAggregationQuery(req *fspb.RunAggregationQueryRequest, stream grpc.ServerStreamingServer[fspb.RunAggregationQueryResponse]) error {
	aq := req.GetStructuredAggregationQuery()
	if aq == nil || aq.GetStructuredQuery() == nil {
		return gcperr.GRPC(gcperr.InvalidArgument, "structured_aggregation_query is required")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	data, err := s.state.dataFor(req.GetTransaction())
	if err != nil {
		return err
	}
	newTxID := []byte(nil)
	if req.GetNewTransaction() != nil {
		newTxID = transactionID()
		s.state.transactions[string(newTxID)] = &transaction{data: cloneData(s.state.documents), active: true}
		data = s.state.transactions[string(newTxID)].data
	}
	// Aggregation is over the underlying query result before projection, so use a copy without select.
	inner := proto.Clone(aq.GetStructuredQuery()).(*fspb.StructuredQuery)
	inner.Select = nil
	docs, err := s.queryDocuments(data, req.GetParent(), inner)
	if err != nil {
		return err
	}
	fields := map[string]*fspb.Value{}
	for i, aggregation := range aq.GetAggregations() {
		alias := aggregation.GetAlias()
		if alias == "" {
			alias = "aggregate_" + strconv.Itoa(i+1)
		}
		if count := aggregation.GetCount(); count != nil {
			n := int64(len(docs))
			if count.GetUpTo() != nil && count.GetUpTo().GetValue() < n {
				n = count.GetUpTo().GetValue()
			}
			fields[alias] = &fspb.Value{ValueType: &fspb.Value_IntegerValue{IntegerValue: n}}
			continue
		}
		var fieldPath string
		isAvg := false
		if aggregation.GetSum() != nil {
			fieldPath = aggregation.GetSum().GetField().GetFieldPath()
		} else if aggregation.GetAvg() != nil {
			fieldPath = aggregation.GetAvg().GetField().GetFieldPath()
			isAvg = true
		} else {
			return gcperr.GRPC(gcperr.InvalidArgument, "aggregation operator is required")
		}
		sum := 0.0
		numericCount := int64(0)
		allInteger := true
		for _, doc := range docs {
			v, ok := fieldValue(doc.GetFields(), fieldPath)
			if !ok {
				continue
			}
			num, numeric, integer := aggregationNumber(v)
			if !numeric {
				continue
			}
			sum += num
			numericCount++
			if !integer {
				allInteger = false
			}
		}
		if isAvg {
			if numericCount == 0 {
				fields[alias] = &fspb.Value{ValueType: &fspb.Value_NullValue{NullValue: 0}}
			} else {
				fields[alias] = &fspb.Value{ValueType: &fspb.Value_DoubleValue{DoubleValue: sum / float64(numericCount)}}
			}
		} else if allInteger {
			fields[alias] = &fspb.Value{ValueType: &fspb.Value_IntegerValue{IntegerValue: int64(sum)}}
		} else {
			fields[alias] = &fspb.Value{ValueType: &fspb.Value_DoubleValue{DoubleValue: sum}}
		}
	}
	return stream.Send(&fspb.RunAggregationQueryResponse{
		Result: &fspb.AggregationResult{AggregateFields: fields}, Transaction: newTxID, ReadTime: timestamppb.Now(),
	})
}

func (s *Server) Write(stream grpc.BidiStreamingServer[fspb.WriteRequest, fspb.WriteResponse]) error {
	first, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	if !validDatabase(first.GetDatabase()) {
		return gcperr.GRPC(gcperr.InvalidArgument, "invalid database")
	}
	streamID := first.GetStreamId()
	if streamID == "" {
		streamID = randomDocumentID()
	}
	streamToken := transactionID()
	if err := stream.Send(&fspb.WriteResponse{StreamId: streamID, StreamToken: streamToken}); err != nil {
		return err
	}
	pending := first
	for {
		if pending == nil {
			pending, err = stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
		req := pending
		pending = nil
		if len(req.GetWrites()) == 0 {
			continue
		}
		if len(req.GetStreamToken()) > 0 && string(req.GetStreamToken()) != string(streamToken) {
			return gcperr.GRPC(gcperr.InvalidArgument, "invalid stream token")
		}
		s.state.mu.Lock()
		candidate := cloneData(s.state.documents)
		now := timestamppb.Now()
		results := make([]*fspb.WriteResult, 0, len(req.GetWrites()))
		var writeErr error
		for _, write := range req.GetWrites() {
			result, err := applyWrite(candidate, write, now)
			if err != nil {
				writeErr = err
				break
			}
			results = append(results, result)
		}
		if writeErr == nil {
			s.state.documents = candidate
		}
		s.state.mu.Unlock()
		if writeErr != nil {
			return writeErr
		}
		if err := stream.Send(&fspb.WriteResponse{StreamId: streamID, StreamToken: streamToken, WriteResults: results, CommitTime: now}); err != nil {
			return err
		}
	}
}

func (s *Server) BeginTransaction(ctx context.Context, req *fspb.BeginTransactionRequest) (*fspb.BeginTransactionResponse, error) {
	if !validDatabase(req.GetDatabase()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid database")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	id := transactionID()
	s.state.transactions[string(id)] = &transaction{data: cloneData(s.state.documents), active: true}
	return &fspb.BeginTransactionResponse{Transaction: id}, nil
}

func (s *Server) Commit(ctx context.Context, req *fspb.CommitRequest) (*fspb.CommitResponse, error) {
	if !validDatabase(req.GetDatabase()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid database")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	data := s.state.documents
	if len(req.GetTransaction()) > 0 {
		tx, ok := s.state.transactions[string(req.GetTransaction())]
		if !ok || !tx.active {
			return nil, gcperr.GRPC(gcperr.InvalidArgument, "transaction is not active")
		}
		data = tx.data
	}
	now := timestamppb.Now()
	results := make([]*fspb.WriteResult, 0, len(req.GetWrites()))
	for _, write := range req.GetWrites() {
		result, err := applyWrite(data, write, now)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if len(req.GetTransaction()) > 0 {
		s.state.documents = data
		delete(s.state.transactions, string(req.GetTransaction()))
	}
	return &fspb.CommitResponse{WriteResults: results, CommitTime: now}, nil
}

func (s *Server) Rollback(ctx context.Context, req *fspb.RollbackRequest) (*emptypb.Empty, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	if _, ok := s.state.transactions[string(req.GetTransaction())]; !ok {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "transaction is not active")
	}
	delete(s.state.transactions, string(req.GetTransaction()))
	return &emptypb.Empty{}, nil
}

func (s *Server) BatchWrite(ctx context.Context, req *fspb.BatchWriteRequest) (*fspb.BatchWriteResponse, error) {
	if !validDatabase(req.GetDatabase()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid database")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	resp := &fspb.BatchWriteResponse{}
	for _, write := range req.GetWrites() {
		result, err := applyWrite(s.state.documents, write, timestamppb.Now())
		if err == nil {
			resp.WriteResults = append(resp.WriteResults, result)
			resp.Status = append(resp.Status, &statuspb.Status{Code: int32(codes.OK)})
		} else {
			st, _ := status.FromError(err)
			resp.WriteResults = append(resp.WriteResults, &fspb.WriteResult{})
			resp.Status = append(resp.Status, &statuspb.Status{Code: int32(st.Code()), Message: st.Message()})
		}
	}
	return resp, nil
}

func (s *Server) ListCollectionIds(ctx context.Context, req *fspb.ListCollectionIdsRequest) (*fspb.ListCollectionIdsResponse, error) {
	idx := strings.Index(req.GetParent(), "/documents")
	if idx < 0 {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid parent")
	}
	database := req.GetParent()[:idx]
	parentRel := parentRelative(req.GetParent())
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	seen := map[string]bool{}
	for path := range s.state.documents[database] {
		parts := strings.Split(path, "/")
		for i := 0; i+1 < len(parts); i++ {
			collectionPrefix := strings.Join(parts[:i], "/")
			if collectionPrefix == parentRel {
				seen[parts[i]] = true
			}
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	start, _ := strconv.Atoi(req.GetPageToken())
	if start > len(ids) {
		start = len(ids)
	}
	size := int(req.GetPageSize())
	if size <= 0 || size > 300 {
		size = 300
	}
	end := start + size
	if end > len(ids) {
		end = len(ids)
	}
	resp := &fspb.ListCollectionIdsResponse{CollectionIds: ids[start:end]}
	if end < len(ids) {
		resp.NextPageToken = strconv.Itoa(end)
	}
	return resp, nil
}
