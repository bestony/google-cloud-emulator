package compute

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	computepb "github.com/bestony/google-cloud-emulator/gen/google/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// genericComputeHandler provides shallow but stateful control-plane behavior
// for the long tail of Compute services. Compute's API is unusually regular:
// resources are scoped by project plus global/zone/region, mutating methods
// return Operation, and CRUD requests carry the resource body in a
// *_resource field. Reflection lets one implementation cover those services
// without weakening the hand-written core resources.
type genericComputeHandler struct {
	state *State

	resourceTypesMu sync.RWMutex
	resourceTypes   map[string]string
}

func newGenericComputeHandler(state *State) *genericComputeHandler {
	return &genericComputeHandler{state: state, resourceTypes: map[string]string{}}
}

func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func singularServiceName(service string) string {
	if strings.HasSuffix(service, "ies") {
		return strings.TrimSuffix(service, "ies") + "y"
	}
	if strings.HasSuffix(service, "sses") {
		return strings.TrimSuffix(service, "es")
	}
	return strings.TrimSuffix(service, "s")
}

func stringField(msg proto.Message, name string) string {
	if msg == nil {
		return ""
	}
	m := msg.ProtoReflect()
	fd := m.Descriptor().Fields().ByName(protoreflect.Name(name))
	if fd == nil || fd.Kind() != protoreflect.StringKind {
		return ""
	}
	return m.Get(fd).String()
}

func setStringField(msg proto.Message, name, value string) {
	m := msg.ProtoReflect()
	fd := m.Descriptor().Fields().ByName(protoreflect.Name(name))
	if fd != nil && fd.Kind() == protoreflect.StringKind {
		m.Set(fd, protoreflect.ValueOfString(value))
	}
}

func setUint64Field(msg proto.Message, name string, value uint64) {
	m := msg.ProtoReflect()
	fd := m.Descriptor().Fields().ByName(protoreflect.Name(name))
	if fd != nil && fd.Kind() == protoreflect.Uint64Kind {
		m.Set(fd, protoreflect.ValueOfUint64(value))
	}
}

func fieldMessage(msg proto.Message, fd protoreflect.FieldDescriptor) proto.Message {
	if fd == nil || fd.Kind() != protoreflect.MessageKind || fd.IsList() || fd.IsMap() {
		return nil
	}
	v := msg.ProtoReflect().Get(fd)
	if !v.IsValid() || v.Message().Interface() == nil {
		return nil
	}
	return v.Message().Interface()
}

func resourceBody(req proto.Message, preferredType string) proto.Message {
	m := req.ProtoReflect()
	fields := m.Descriptor().Fields()
	var fallback proto.Message
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.Kind() != protoreflect.MessageKind || fd.IsList() || fd.IsMap() || !m.Has(fd) {
			continue
		}
		candidate := fieldMessage(req, fd)
		if candidate == nil {
			continue
		}
		typeName := string(candidate.ProtoReflect().Descriptor().Name())
		fieldName := string(fd.Name())
		if typeName == preferredType {
			return candidate
		}
		if strings.HasSuffix(fieldName, "_resource") && fallback == nil {
			fallback = candidate
		}
	}
	return fallback
}

func serviceResourceType(h *genericComputeHandler, service string, req, resp proto.Message) string {
	if resp != nil {
		name := string(resp.ProtoReflect().Descriptor().Name())
		if name != "Operation" && !strings.HasSuffix(name, "List") && !strings.HasSuffix(name, "AggregatedList") {
			return name
		}
	}
	h.resourceTypesMu.RLock()
	if cached := h.resourceTypes[service]; cached != "" {
		h.resourceTypesMu.RUnlock()
		return cached
	}
	h.resourceTypesMu.RUnlock()
	inferred := singularServiceName(service)
	if body := resourceBody(req, inferred); body != nil {
		return string(body.ProtoReflect().Descriptor().Name())
	}
	return inferred
}

func computeScope(req proto.Message) (project, scope, scopePath string) {
	project = stringField(req, "project")
	if project == "" {
		project = stringField(req, "resource_project")
	}
	if zone := zoneName(stringField(req, "zone")); zone != "" {
		return project, "zones/" + zone, "projects/" + project + "/zones/" + zone
	}
	if region := regionName(stringField(req, "region")); region != "" {
		return project, "regions/" + region, "projects/" + project + "/regions/" + region
	}
	if location := stringField(req, "location"); location != "" {
		return project, "locations/" + location, "projects/" + project + "/locations/" + location
	}
	return project, "global", "projects/" + project
}

var nonNameComputeFields = map[string]bool{
	"project": true, "resource_project": true, "zone": true, "region": true, "location": true,
	"request_id": true, "filter": true, "page_token": true, "order_by": true,
	"max_results": true, "return_partial_success": true, "parent": true, "service_project_number": true,
}

func inferResourceName(req proto.Message, resourceType string) string {
	exact := camelToSnake(resourceType)
	if v := stringField(req, exact); v != "" {
		return lastPathSegment(v)
	}
	if v := stringField(req, "name"); v != "" {
		return lastPathSegment(v)
	}
	m := req.ProtoReflect()
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.Kind() != protoreflect.StringKind || fd.IsList() || nonNameComputeFields[string(fd.Name())] {
			continue
		}
		name := string(fd.Name())
		if name == exact || strings.HasSuffix(name, "_"+exact) {
			if v := m.Get(fd).String(); v != "" {
				return lastPathSegment(v)
			}
		}
	}
	found := ""
	count := 0
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.Kind() != protoreflect.StringKind || fd.IsList() || nonNameComputeFields[string(fd.Name())] {
			continue
		}
		if v := m.Get(fd).String(); v != "" {
			found = v
			count++
		}
	}
	if count == 1 {
		return lastPathSegment(found)
	}
	return ""
}

func lastPathSegment(s string) string {
	s = strings.TrimSuffix(s, "/")
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

func genericKey(service, project, scope, name string) string {
	return strings.Join([]string{service, project, scope, name}, "|")
}

func computeSelfLink(project, scope, resourceType, name string) string {
	kind := camelToSnake(resourceType)
	collection := kind + "s"
	if strings.HasSuffix(kind, "y") {
		collection = strings.TrimSuffix(kind, "y") + "ies"
	}
	if strings.HasSuffix(kind, "s") {
		collection = kind + "es"
	}
	if scope == "global" {
		return fmt.Sprintf("%s/projects/%s/%s/%s", selfLinkBase, project, collection, name)
	}
	parts := strings.Split(scope, "/")
	return fmt.Sprintf("%s/projects/%s/%s/%s/%s/%s", selfLinkBase, project, parts[0], parts[1], collection, name)
}

func populateResourceDefaults(msg proto.Message, project, scope, resourceType string, id uint64) {
	name := stringField(msg, "name")
	if stringField(msg, "self_link") == "" && name != "" {
		setStringField(msg, "self_link", computeSelfLink(project, scope, resourceType, name))
	}
	if stringField(msg, "creation_timestamp") == "" {
		setStringField(msg, "creation_timestamp", rfc3339Now())
	}
	if stringField(msg, "kind") == "" {
		setStringField(msg, "kind", "compute#"+lowerFirst(resourceType))
	}
	m := msg.ProtoReflect()
	if fd := m.Descriptor().Fields().ByName("id"); fd != nil && !m.Has(fd) {
		setUint64Field(msg, "id", id)
	}
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func (h *genericComputeHandler) operationResponse(scopePath, opType, target string, resp proto.Message) {
	h.state.mu.Lock()
	op := h.state.addOperation(scopePath, opType, target)
	h.state.mu.Unlock()
	proto.Merge(resp, op)
}

func (h *genericComputeHandler) handleGet(service string, req, resp proto.Message) error {
	project, scope, _ := computeScope(req)
	resourceType := serviceResourceType(h, service, req, resp)
	name := inferResourceName(req, resourceType)
	if project == "" || name == "" {
		return invalidArg("project and resource name are required")
	}
	key := genericKey(service, project, scope, name)
	h.state.mu.RLock()
	stored := h.state.genericResources[key]
	h.state.mu.RUnlock()
	if stored == nil {
		return notFound(lowerFirst(resourceType), project, name)
	}
	proto.Merge(resp, stored)
	return nil
}

func matchGenericKey(key, service, project, scope string) bool {
	parts := strings.Split(key, "|")
	if len(parts) != 4 || parts[0] != service || parts[1] != project {
		return false
	}
	return scope == "" || parts[2] == scope
}

func (h *genericComputeHandler) storedResources(service, project, scope string) []proto.Message {
	h.state.mu.RLock()
	defer h.state.mu.RUnlock()
	var out []proto.Message
	for key, msg := range h.state.genericResources {
		if matchGenericKey(key, service, project, scope) {
			out = append(out, proto.Clone(msg))
		}
	}
	sort.Slice(out, func(i, j int) bool { return stringField(out[i], "name") < stringField(out[j], "name") })
	return out
}

func (h *genericComputeHandler) handleList(service string, req, resp proto.Message) error {
	project, scope, _ := computeScope(req)
	if project == "" {
		return invalidArg("project is required")
	}
	items := h.storedResources(service, project, scope)
	m := resp.ProtoReflect()
	fd := m.Descriptor().Fields().ByName("items")
	if fd == nil || !fd.IsList() || fd.Kind() != protoreflect.MessageKind {
		return nil
	}
	list := m.Mutable(fd).List()
	for _, item := range items {
		list.Append(protoreflect.ValueOfMessage(item.ProtoReflect()))
	}
	return nil
}

func (h *genericComputeHandler) handleAggregatedList(service string, req, resp proto.Message) error {
	project, _, _ := computeScope(req)
	if project == "" {
		return invalidArg("project is required")
	}
	m := resp.ProtoReflect()
	fd := m.Descriptor().Fields().ByName("items")
	if fd == nil || !fd.IsMap() || fd.MapValue().Kind() != protoreflect.MessageKind {
		return nil
	}
	valueDesc := fd.MapValue().Message()
	var valuesField protoreflect.FieldDescriptor
	for i := 0; i < valueDesc.Fields().Len(); i++ {
		candidate := valueDesc.Fields().Get(i)
		if candidate.IsList() && candidate.Kind() == protoreflect.MessageKind {
			valuesField = candidate
			break
		}
	}
	if valuesField == nil {
		return nil
	}
	grouped := map[string][]proto.Message{}
	h.state.mu.RLock()
	for key, item := range h.state.genericResources {
		if matchGenericKey(key, service, project, "") {
			scope := strings.Split(key, "|")[2]
			grouped[scope] = append(grouped[scope], proto.Clone(item))
		}
	}
	h.state.mu.RUnlock()
	mapValue := m.Mutable(fd).Map()
	for scope, resources := range grouped {
		scoped := dynamicpb.NewMessage(valueDesc)
		list := scoped.Mutable(valuesField).List()
		for _, item := range resources {
			list.Append(protoreflect.ValueOfMessage(item.ProtoReflect()))
		}
		mapValue.Set(protoreflect.ValueOfString(scope).MapKey(), protoreflect.ValueOfMessage(scoped))
	}
	return nil
}

func copyPopulatedFields(dst, src proto.Message) {
	sm := src.ProtoReflect()
	dm := dst.ProtoReflect()
	fields := sm.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if sm.Has(fd) {
			dm.Set(fd, sm.Get(fd))
		}
	}
}

func (h *genericComputeHandler) handleInsert(service, method string, req, resp proto.Message) error {
	project, scope, scopePath := computeScope(req)
	preferred := singularServiceName(service)
	body := resourceBody(req, preferred)
	if project == "" {
		return invalidArg("project is required")
	}
	if body == nil {
		return invalidArg("resource body is required")
	}
	resourceType := string(body.ProtoReflect().Descriptor().Name())
	name := stringField(body, "name")
	if name == "" {
		return invalidArg("resource name is required")
	}
	stored := proto.Clone(body)
	populateResourceDefaults(stored, project, scope, resourceType, h.state.nextID())
	key := genericKey(service, project, scope, name)
	h.state.mu.Lock()
	if h.state.genericResources[key] != nil {
		h.state.mu.Unlock()
		return alreadyExists(lowerFirst(resourceType), project, name)
	}
	h.state.genericResources[key] = stored
	op := h.state.addOperation(scopePath, method, stringField(stored, "self_link"))
	h.state.mu.Unlock()
	h.resourceTypesMu.Lock()
	h.resourceTypes[service] = resourceType
	h.resourceTypesMu.Unlock()
	proto.Merge(resp, op)
	return nil
}

func (h *genericComputeHandler) handleDelete(service, method string, req, resp proto.Message) error {
	project, scope, scopePath := computeScope(req)
	resourceType := serviceResourceType(h, service, req, nil)
	name := inferResourceName(req, resourceType)
	if project == "" || name == "" {
		return invalidArg("project and resource name are required")
	}
	key := genericKey(service, project, scope, name)
	h.state.mu.Lock()
	stored := h.state.genericResources[key]
	if stored == nil {
		h.state.mu.Unlock()
		return notFound(lowerFirst(resourceType), project, name)
	}
	delete(h.state.genericResources, key)
	op := h.state.addOperation(scopePath, method, stringField(stored, "self_link"))
	h.state.mu.Unlock()
	proto.Merge(resp, op)
	return nil
}

func (h *genericComputeHandler) handleUpdate(service, method string, req, resp proto.Message, patch bool) error {
	project, scope, scopePath := computeScope(req)
	preferred := serviceResourceType(h, service, req, nil)
	body := resourceBody(req, preferred)
	resourceType := preferred
	if body != nil {
		resourceType = string(body.ProtoReflect().Descriptor().Name())
	}
	name := inferResourceName(req, resourceType)
	if name == "" && body != nil {
		name = stringField(body, "name")
	}
	if project == "" || name == "" || body == nil {
		return invalidArg("project, resource name and resource body are required")
	}
	key := genericKey(service, project, scope, name)
	h.state.mu.Lock()
	existing := h.state.genericResources[key]
	if existing == nil {
		h.state.mu.Unlock()
		return notFound(lowerFirst(resourceType), project, name)
	}
	var updated proto.Message
	if patch {
		updated = proto.Clone(existing)
		copyPopulatedFields(updated, body)
	} else {
		updated = proto.Clone(body)
		if stringField(updated, "self_link") == "" {
			setStringField(updated, "self_link", stringField(existing, "self_link"))
		}
		if stringField(updated, "creation_timestamp") == "" {
			setStringField(updated, "creation_timestamp", stringField(existing, "creation_timestamp"))
		}
	}
	populateResourceDefaults(updated, project, scope, resourceType, h.state.nextID())
	h.state.genericResources[key] = updated
	op := h.state.addOperation(scopePath, method, stringField(updated, "self_link"))
	h.state.mu.Unlock()
	h.resourceTypesMu.Lock()
	h.resourceTypes[service] = resourceType
	h.resourceTypesMu.Unlock()
	proto.Merge(resp, op)
	return nil
}

func isOperationResponse(resp proto.Message) bool {
	return resp != nil && string(resp.ProtoReflect().Descriptor().FullName()) == "google.cloud.compute.v1.Operation"
}

func (h *genericComputeHandler) handleCustom(service, method string, req, resp proto.Message) error {
	project, scope, scopePath := computeScope(req)
	resourceType := serviceResourceType(h, service, req, nil)
	name := inferResourceName(req, resourceType)
	target := ""
	if project != "" && name != "" {
		key := genericKey(service, project, scope, name)
		h.state.mu.RLock()
		stored := h.state.genericResources[key]
		h.state.mu.RUnlock()
		if stored == nil {
			return notFound(lowerFirst(resourceType), project, name)
		}
		target = stringField(stored, "self_link")
		if body := resourceBody(req, resourceType); body != nil && string(body.ProtoReflect().Descriptor().Name()) == resourceType {
			updated := proto.Clone(stored)
			copyPopulatedFields(updated, body)
			h.state.mu.Lock()
			h.state.genericResources[key] = updated
			h.state.mu.Unlock()
		}
	}
	if isOperationResponse(resp) {
		h.operationResponse(scopePath, method, target, resp)
	} else if name != "" {
		h.state.mu.RLock()
		stored := h.state.genericResources[genericKey(service, project, scope, name)]
		h.state.mu.RUnlock()
		if stored != nil && stored.ProtoReflect().Descriptor() == resp.ProtoReflect().Descriptor() {
			proto.Merge(resp, stored)
		}
	}
	return nil
}

func (h *genericComputeHandler) Handle(ctx context.Context, service, method string, req, resp proto.Message) error {
	_ = ctx
	switch method {
	case "Get":
		return h.handleGet(service, req, resp)
	case "List":
		return h.handleList(service, req, resp)
	case "AggregatedList":
		return h.handleAggregatedList(service, req, resp)
	case "Insert":
		return h.handleInsert(service, method, req, resp)
	case "Delete":
		return h.handleDelete(service, method, req, resp)
	case "Patch":
		return h.handleUpdate(service, method, req, resp, true)
	case "Update":
		return h.handleUpdate(service, method, req, resp, false)
	default:
		return h.handleCustom(service, method, req, resp)
	}
}

var _ = computepb.Operation{}
