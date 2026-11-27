// Package run emulates google.cloud.run.v2.Services over gRPC and REST
// (via grpc-gateway). State is in-memory only.
//
// M1 scope: services and revisions control plane, including LRO-backed
// create/update/delete. Jobs, executions, instances, worker pools and
// build sources return UNIMPLEMENTED.
package run

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	runpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/run/v2"
	api "github.com/bestony/google-cloud-emulator/gen/google/api"
	"github.com/bestony/google-cloud-emulator/core/iam"
	"github.com/bestony/google-cloud-emulator/core/lro"
	"github.com/bestony/google-cloud-emulator/core/pagination"
)

var (
	locationRe  = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+$`)
	serviceRe   = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+/services/[^/]+$`)
	revisionRe  = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+/services/[^/]+/revisions/[^/]+$`)
	serviceIDRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

// serviceEntry pairs a service with its revision counter.
type serviceEntry struct {
	Svc         *runpb.Service
	RevCounter  int
}

// State holds all Cloud Run resources in memory.
type State struct {
	mu        sync.RWMutex
	services  map[string]*serviceEntry
	revisions map[string]*runpb.Revision
	lro       *lro.Store
	iam       *iam.Store
}

// NewState returns an empty State backed by the shared LRO and IAM stores.
func NewState(lroStore *lro.Store, iamStore *iam.Store) *State {
	return &State{
		services:  make(map[string]*serviceEntry),
		revisions: make(map[string]*runpb.Revision),
		lro:       lroStore,
		iam:       iamStore,
	}
}

// LROStore exposes the shared LRO store.
func (s *State) LROStore() *lro.Store { return s.lro }

// IAMStore exposes the shared IAM store.
func (s *State) IAMStore() *iam.Store { return s.iam }

func now() *timestamppb.Timestamp { return timestamppb.New(time.Now().UTC()) }

func newUID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneDuration(in *durationpb.Duration) *durationpb.Duration {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*durationpb.Duration)
}

func cloneContainers(in []*runpb.Container) []*runpb.Container {
	if in == nil {
		return nil
	}
	out := make([]*runpb.Container, len(in))
	for i, c := range in {
		out[i] = proto.Clone(c).(*runpb.Container)
	}
	return out
}

func cloneVolumes(in []*runpb.Volume) []*runpb.Volume {
	if in == nil {
		return nil
	}
	out := make([]*runpb.Volume, len(in))
	for i, v := range in {
		out[i] = proto.Clone(v).(*runpb.Volume)
	}
	return out
}

func cloneTrafficTargets(in []*runpb.TrafficTarget) []*runpb.TrafficTarget {
	if in == nil {
		return nil
	}
	out := make([]*runpb.TrafficTarget, len(in))
	for i, t := range in {
		out[i] = proto.Clone(t).(*runpb.TrafficTarget)
	}
	return out
}

func cloneBinaryAuthorization(in *runpb.BinaryAuthorization) *runpb.BinaryAuthorization {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*runpb.BinaryAuthorization)
}

func cloneServiceScaling(in *runpb.ServiceScaling) *runpb.ServiceScaling {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*runpb.ServiceScaling)
}

func cloneRevisionScaling(in *runpb.RevisionScaling) *runpb.RevisionScaling {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*runpb.RevisionScaling)
}

func cloneRevisionTemplate(in *runpb.RevisionTemplate) *runpb.RevisionTemplate {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*runpb.RevisionTemplate)
}

// readyCondition returns a succeeded Ready condition.
func readyCondition() *runpb.Condition {
	return &runpb.Condition{
		Type:    "Ready",
		State:   runpb.Condition_CONDITION_SUCCEEDED,
		Message: "Service is ready",
	}
}

// revisionNameFor returns the revision resource name for a service and counter.
func revisionNameFor(serviceName string, counter int) string {
	return fmt.Sprintf("%s/revisions/%s-%05d", serviceName, serviceName[strings.LastIndex(serviceName, "/")+1:], counter)
}

// buildRevision materializes a Revision from a service template.
func buildRevision(serviceName string, counter int, tmpl *runpb.RevisionTemplate, svcLabels, svcAnnotations map[string]string) *runpb.Revision {
	rev := &runpb.Revision{
		Name:                       revisionNameFor(serviceName, counter),
		Uid:                        newUID(),
		Generation:                 1,
		Service:                    serviceName,
		CreateTime:                 now(),
		UpdateTime:                 now(),
		LaunchStage:                api.LaunchStage_GA,
		ObservedGeneration:         1,
		Labels:                     cloneStringMap(tmpl.GetLabels()),
		Annotations:                cloneStringMap(tmpl.GetAnnotations()),
		Scaling:                    cloneRevisionScaling(tmpl.Scaling),
		Timeout:                    cloneDuration(tmpl.Timeout),
		ServiceAccount:             tmpl.GetServiceAccount(),
		Containers:                 cloneContainers(tmpl.Containers),
		Volumes:                    cloneVolumes(tmpl.Volumes),
		MaxInstanceRequestConcurrency: tmpl.GetMaxInstanceRequestConcurrency(),
		ExecutionEnvironment:        tmpl.GetExecutionEnvironment(),
		SessionAffinity:             tmpl.GetSessionAffinity(),
		Conditions:                  []*runpb.Condition{readyCondition()},
	}
	if len(rev.Labels) == 0 {
		rev.Labels = cloneStringMap(svcLabels)
	}
	if len(rev.Annotations) == 0 {
		rev.Annotations = cloneStringMap(svcAnnotations)
	}
	return rev
}

// CreateService creates a service and its first revision; code "" or error code.
func (s *State) CreateService(parent, serviceID string, svc *runpb.Service) (*runpb.Service, string) {
	if !locationRe.MatchString(parent) {
		return nil, "invalid-parent"
	}
	if !serviceIDRe.MatchString(serviceID) || len(serviceID) > 63 {
		return nil, "invalid-name"
	}
	name := parent + "/services/" + serviceID
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[name]; ok {
		return nil, "already-exists"
	}
	entry := &serviceEntry{RevCounter: 1}
	protoSvc := proto.Clone(svc).(*runpb.Service)
	protoSvc.Name = name
	protoSvc.Uid = newUID()
	protoSvc.Generation = 1
	protoSvc.CreateTime = now()
	protoSvc.UpdateTime = protoSvc.CreateTime
	protoSvc.ObservedGeneration = 1
	protoSvc.LaunchStage = api.LaunchStage_GA
	protoSvc.TerminalCondition = readyCondition()
	protoSvc.Conditions = []*runpb.Condition{readyCondition()}
	if protoSvc.GetTemplate() == nil {
		protoSvc.Template = &runpb.RevisionTemplate{}
	}
	entry.Svc = protoSvc
	s.services[name] = entry
	s.addRevisionLocked(entry)
	return proto.Clone(entry.Svc).(*runpb.Service), ""
}

// addRevisionLocked materializes the next revision and points the service at it.
func (s *State) addRevisionLocked(entry *serviceEntry) string {
	entry.RevCounter++
	rev := buildRevision(entry.Svc.Name, entry.RevCounter, entry.Svc.GetTemplate(), entry.Svc.GetLabels(), entry.Svc.GetAnnotations())
	s.revisions[rev.Name] = rev
	entry.Svc.LatestCreatedRevision = rev.Name
	entry.Svc.LatestReadyRevision = rev.Name
	entry.Svc.ObservedGeneration = entry.Svc.Generation
	entry.Svc.UpdateTime = now()
	entry.Svc.Uri = "https://" + entry.Svc.Name[strings.LastIndex(entry.Svc.Name, "/")+1:] + ".run.app"
	return ""
}

// GetService returns a copy or nil.
func (s *State) GetService(name string) *runpb.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.services[name]
	if !ok {
		return nil
	}
	return proto.Clone(entry.Svc).(*runpb.Service)
}

// ListServices returns a sorted page under parent (a location name).
func (s *State) ListServices(parent string, pageSize int, pageToken string) ([]*runpb.Service, string, string) {
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", "invalid-page-token"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var names []string
	prefix := strings.TrimSuffix(parent, "/") + "/services/"
	for name := range s.services {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	total := len(names)
	if offset >= total {
		return []*runpb.Service{}, "", ""
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	out := make([]*runpb.Service, 0, end-offset)
	for _, n := range names[offset:end] {
		out = append(out, proto.Clone(s.services[n].Svc).(*runpb.Service))
	}
	next := ""
	if end < total {
		next = pagination.Encode(end)
	}
	return out, next, ""
}

// serviceMutableFields are the UpdateService mask paths the emulator supports.
var serviceMutableFields = map[string]bool{
	"labels": true, "annotations": true, "template": true, "traffic": true,
	"ingress": true, "launch_stage": true, "binary_authorization": true,
	"custom_audiences": true, "invoker_iam_disabled": true, "default_uri_disabled": true,
	"description": true, "scaling": true, "iap_enabled": true, "ssh_enabled": true,
}

// UpdateService applies whitelisted mask paths and rolls a new revision when the
// template changes. Returns the updated service or an error code.
func (s *State) UpdateService(patch *runpb.Service, maskPaths []string, forceNewRevision bool) (*runpb.Service, string) {
	name := patch.GetName()
	if !serviceRe.MatchString(name) {
		return nil, "invalid-name"
	}
	if len(maskPaths) == 0 {
		return nil, "invalid-mask:"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.services[name]
	if !ok {
		return nil, "not-found"
	}
	pc := proto.Clone(patch).(*runpb.Service)
	templateChanged := false
	for _, p := range maskPaths {
		switch p {
		case "labels":
			entry.Svc.Labels = cloneStringMap(pc.Labels)
		case "annotations":
			entry.Svc.Annotations = cloneStringMap(pc.Annotations)
		case "template":
			entry.Svc.Template = cloneRevisionTemplate(pc.Template)
			templateChanged = true
		case "traffic":
			entry.Svc.Traffic = cloneTrafficTargets(pc.Traffic)
		case "ingress":
			entry.Svc.Ingress = pc.Ingress
		case "launch_stage":
			entry.Svc.LaunchStage = pc.LaunchStage
		case "binary_authorization":
			entry.Svc.BinaryAuthorization = cloneBinaryAuthorization(pc.BinaryAuthorization)
		case "custom_audiences":
			entry.Svc.CustomAudiences = append([]string(nil), pc.CustomAudiences...)
		case "invoker_iam_disabled":
			entry.Svc.InvokerIamDisabled = pc.InvokerIamDisabled
		case "default_uri_disabled":
			entry.Svc.DefaultUriDisabled = pc.DefaultUriDisabled
		case "description":
			entry.Svc.Description = pc.Description
		case "scaling":
			entry.Svc.Scaling = cloneServiceScaling(pc.Scaling)
		case "iap_enabled":
			entry.Svc.IapEnabled = pc.IapEnabled
		case "ssh_enabled":
			entry.Svc.SshEnabled = pc.SshEnabled
		default:
			return nil, "invalid-mask:" + p
		}
	}
	entry.Svc.Generation++
	entry.Svc.UpdateTime = now()
	if templateChanged || forceNewRevision {
		s.addRevisionLocked(entry)
	}
	entry.Svc.ObservedGeneration = entry.Svc.Generation
	return proto.Clone(entry.Svc).(*runpb.Service), ""
}

// DeleteService removes a service and its revisions; returns the deleted service.
func (s *State) DeleteService(name string) *runpb.Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.services[name]
	if !ok {
		return nil
	}
	prefix := name + "/revisions/"
	for revName := range s.revisions {
		if strings.HasPrefix(revName, prefix) {
			delete(s.revisions, revName)
		}
	}
	out := proto.Clone(entry.Svc).(*runpb.Service)
	delete(s.services, name)
	return out
}

// GetRevision returns a copy or nil.
func (s *State) GetRevision(name string) *runpb.Revision {
	if !revisionRe.MatchString(name) {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rev, ok := s.revisions[name]
	if !ok {
		return nil
	}
	return proto.Clone(rev).(*runpb.Revision)
}

// ListRevisions returns a sorted page of revisions. parent may be a location
// (projects/p/locations/l), a service, or a revisions collection.
func (s *State) ListRevisions(parent string, pageSize int, pageToken string) ([]*runpb.Revision, string, string) {
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", "invalid-page-token"
	}
	base := strings.TrimSuffix(parent, "/")
	var prefix string
	switch {
	case strings.HasSuffix(base, "/revisions"):
		prefix = base + "/"
	case serviceRe.MatchString(base):
		prefix = base + "/revisions/"
	case locationRe.MatchString(base):
		prefix = base + "/services/"
		// match revisions of every service under the location: use name substring instead
	default:
		return nil, "", "invalid-parent"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var names []string
	for name := range s.revisions {
		if prefix == base+"/services/" {
			// location-level: revision names embed the service path after the location
			if strings.Contains(name, base+"/services/") {
				names = append(names, name)
			}
			continue
		}
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	total := len(names)
	if offset >= total {
		return []*runpb.Revision{}, "", ""
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	out := make([]*runpb.Revision, 0, end-offset)
	for _, n := range names[offset:end] {
		out = append(out, proto.Clone(s.revisions[n]).(*runpb.Revision))
	}
	next := ""
	if end < total {
		next = pagination.Encode(end)
	}
	return out, next, ""
}
