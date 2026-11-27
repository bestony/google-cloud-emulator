// Package secretmanager emulates google.cloud.secretmanager.v1 over gRPC and
// REST (via grpc-gateway). State is in-memory only.
package secretmanager

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bestony/google-cloud-emulator/core/iam"
	"github.com/bestony/google-cloud-emulator/core/pagination"
	smpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/secretmanager/v1"
)

var (
	projectParentRe  = regexp.MustCompile(`^projects/[^/]+$`)
	locationParentRe = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+$`)
	secretNameRe     = regexp.MustCompile(`^projects/[^/]+/secrets/[^/]+$`)
	secretNameLocRe  = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+/secrets/[^/]+$`)
	versionSuffixRe  = regexp.MustCompile(`^(.+)/versions/([^/]+)$`)
	secretIDRe       = regexp.MustCompile(`^[A-Za-z0-9_-]{1,255}$`)
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

// ChecksumCrc32C computes the CRC32C checksum Google stores for payloads.
func ChecksumCrc32C(data []byte) uint32 { return crc32.Checksum(data, crc32cTable) }

// Version is one secret version's data and lifecycle state.
type Version struct {
	Number      int64
	State       smpb.SecretVersion_State
	CreateTime  time.Time
	DestroyTime *time.Time
	Data        []byte
	Checksum    uint32
}

// Secret is one secret resource with its versions.
type Secret struct {
	Proto       *smpb.Secret
	Versions    map[int64]*Version
	NextVersion int64
	Revision    uint64
}

// State holds all secrets, keyed by full secret resource name.
type State struct {
	mu      sync.RWMutex
	secrets map[string]*Secret
	iam     *iam.Store
}

// NewState returns an empty State backed by the shared IAM store.
func NewState(iamStore *iam.Store) *State {
	return &State{secrets: make(map[string]*Secret), iam: iamStore}
}

// IAMStore returns the shared IAM store.
func (s *State) IAMStore() *iam.Store { return s.iam }

// ValidParent reports whether parent is a valid CreateSecret/ListSecrets parent.
func ValidParent(parent string) bool {
	return projectParentRe.MatchString(parent) || locationParentRe.MatchString(parent)
}

// ValidSecretID reports whether id is a valid secret ID.
func ValidSecretID(id string) bool { return secretIDRe.MatchString(id) }

// SecretResourceName returns the resource name for parent+id.
func SecretResourceName(parent, id string) string { return parent + "/secrets/" + id }

// ParseSecretName validates a full secret resource name (global or location).
func ParseSecretName(name string) bool {
	return secretNameRe.MatchString(name) || secretNameLocRe.MatchString(name)
}

// ParseVersionName splits a version resource name into secret name and version
// identifier (number or alias).
func ParseVersionName(name string) (secretName, versionID string, ok bool) {
	m := versionSuffixRe.FindStringSubmatch(name)
	if m == nil || !ParseSecretName(m[1]) {
		return "", "", false
	}
	return m[1], m[2], true
}

// ResolveVersion returns the version number for versionID ("latest", an alias,
// or a number) on secret sec.
func (sec *Secret) ResolveVersion(versionID string) (int64, bool) {
	if n, err := strconv.ParseInt(versionID, 10, 64); err == nil {
		if _, ok := sec.Versions[n]; ok {
			return n, true
		}
		return 0, false
	}
	if v, ok := sec.Proto.VersionAliases[versionID]; ok {
		if _, ok := sec.Versions[v]; ok {
			return v, true
		}
	}
	return 0, false
}

func now() time.Time { return time.Now().UTC() }

func etagOf(sec *Secret) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:%d", sec.Proto.Name, sec.Revision)))
	return hex.EncodeToString(sum[:])[:16]
}

func versionProto(secName string, v *Version) *smpb.SecretVersion {
	out := &smpb.SecretVersion{
		Name:       fmt.Sprintf("%s/versions/%d", secName, v.Number),
		CreateTime: timestamppb.New(v.CreateTime),
		State:      v.State,
	}
	if v.DestroyTime != nil {
		out.DestroyTime = timestamppb.New(*v.DestroyTime)
	}
	return out
}

// ListSecrets returns one page of secret protos (sorted by name) under parent.
func (s *State) ListSecrets(parent string, pageSize int, pageToken string) ([]*smpb.Secret, string, int, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := parent + "/secrets/"
	var names []string
	for name := range s.secrets {
		if strings.HasPrefix(name, prefix) && !strings.Contains(name[len(prefix):], "/") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	total := len(names)
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	out := make([]*smpb.Secret, 0, end-offset)
	for _, n := range names[offset:end] {
		sec := s.secrets[n]
		cp := proto.Clone(sec.Proto).(*smpb.Secret)
		cp.Etag = etagOf(sec)
		out = append(out, cp)
	}
	next := pagination.PageToken(offset, pageSize, total)
	return out, next, total, ""
}

// CreateSecret creates a secret. tmpl provides mutable fields (labels,
// replication, rotation, ...). Returns ALREADY_EXISTS code on conflict.
func (s *State) CreateSecret(parent, id string, tmpl *smpb.Secret) (*smpb.Secret, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := SecretResourceName(parent, id)
	if _, exists := s.secrets[name]; exists {
		return nil, "already-exists"
	}
	sec := &Secret{
		Proto: &smpb.Secret{
			Name:                      name,
			Replication:               tmpl.GetReplication(),
			Labels:                    cloneStringMap(tmpl.Labels),
			Topics:                    cloneTopics(tmpl.Topics),
			Rotation:                  cloneRotation(tmpl.Rotation),
			VersionAliases:            map[string]int64{},
			Annotations:               cloneStringMap(tmpl.Annotations),
			VersionDestroyTtl:         tmpl.VersionDestroyTtl,
			CustomerManagedEncryption: tmpl.CustomerManagedEncryption,
			Tags:                      cloneStringMap(tmpl.Tags),
			SecretType:                tmpl.SecretType,
			State:                     smpb.Secret_STATE_ACTIVE,
			CreateTime:                timestamppb.New(now()),
		},
		Versions:    map[int64]*Version{},
		NextVersion: 1,
		Revision:    1,
	}
	s.secrets[name] = sec
	cp := proto.Clone(sec.Proto).(*smpb.Secret)
	cp.Etag = etagOf(sec)
	return cp, ""
}

// GetSecret returns a clone of the secret proto.
func (s *State) GetSecret(name string) (*smpb.Secret, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sec, ok := s.secrets[name]
	if !ok {
		return nil, false
	}
	cp := proto.Clone(sec.Proto).(*smpb.Secret)
	cp.Etag = etagOf(sec)
	return cp, true
}

// UpdateSecret applies patch fields (restricted by mask paths when non-empty)
// to the named secret.
func (s *State) UpdateSecret(name string, patch *smpb.Secret, maskPaths map[string]bool) (*smpb.Secret, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sec, ok := s.secrets[name]
	if !ok {
		return nil, "not-found"
	}
	mutable := func(field string) bool {
		return len(maskPaths) == 0 || maskPaths[field]
	}
	p := sec.Proto
	if mutable("labels") && patch.Labels != nil {
		p.Labels = cloneStringMap(patch.Labels)
	}
	if mutable("annotations") && patch.Annotations != nil {
		p.Annotations = cloneStringMap(patch.Annotations)
	}
	if mutable("tags") && patch.Tags != nil {
		p.Tags = cloneStringMap(patch.Tags)
	}
	if mutable("topics") && patch.Topics != nil {
		p.Topics = cloneTopics(patch.Topics)
	}
	if mutable("rotation") && patch.Rotation != nil {
		p.Rotation = cloneRotation(patch.Rotation)
	}
	if mutable("version_destroy_ttl") && patch.VersionDestroyTtl != nil {
		p.VersionDestroyTtl = patch.VersionDestroyTtl
	}
	if mutable("replication") && patch.Replication != nil {
		// Replication is documented immutable; reject changes.
		return nil, "replication-immutable"
	}
	sec.Revision++
	cp := proto.Clone(p).(*smpb.Secret)
	cp.Etag = etagOf(sec)
	return cp, ""
}

// DeleteSecret removes a secret and all its versions.
func (s *State) DeleteSecret(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.secrets[name]; !ok {
		return false
	}
	delete(s.secrets, name)
	return true
}

// AddVersion appends a new ENABLED version with the payload data.
func (s *State) AddVersion(secretName string, payload *smpb.SecretPayload) (*smpb.SecretVersion, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sec, ok := s.secrets[secretName]
	if !ok {
		return nil, "not-found"
	}
	data := payload.GetData()
	if payload.DataCrc32C != nil {
		if got := ChecksumCrc32C(data); got != uint32(*payload.DataCrc32C) {
			return nil, "checksum-mismatch"
		}
	}
	num := sec.NextVersion
	sec.NextVersion++
	v := &Version{
		Number:     num,
		State:      smpb.SecretVersion_ENABLED,
		CreateTime: now(),
		Data:       append([]byte(nil), data...),
		Checksum:   ChecksumCrc32C(data),
	}
	sec.Versions[num] = v
	if sec.Proto.VersionAliases == nil {
		sec.Proto.VersionAliases = map[string]int64{}
	}
	sec.Proto.VersionAliases["latest"] = num
	sec.Revision++
	return versionProto(secretName, v), ""
}

// ListVersions returns one page of version protos for secretName.
// filter may be "" or "state=ENABLED|DISABLED|DESTROYED".
func (s *State) ListVersions(secretName, filter string, pageSize int, pageToken string) ([]*smpb.SecretVersion, string, int, string) {
	wantState, ferr := parseStateFilter(filter)
	if ferr != "" {
		return nil, "", 0, ferr
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	sec, ok := s.secrets[secretName]
	if !ok {
		return nil, "", 0, "not-found"
	}
	var nums []int64
	for n, v := range sec.Versions {
		if wantState == smpb.SecretVersion_STATE_UNSPECIFIED || v.State == wantState {
			nums = append(nums, n)
		}
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] > nums[j] }) // newest first
	total := len(nums)
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	out := make([]*smpb.SecretVersion, 0, end-offset)
	for _, n := range nums[offset:end] {
		out = append(out, versionProto(secretName, sec.Versions[n]))
	}
	next := pagination.PageToken(offset, pageSize, total)
	return out, next, total, ""
}

// GetVersion resolves a version resource name and returns its proto plus the
// internal record (nil data for DESTROYED versions).
func (s *State) GetVersion(name string) (*smpb.SecretVersion, *Version, string) {
	secretName, versionID, ok := ParseVersionName(name)
	if !ok {
		return nil, nil, "invalid-name"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	sec, ok := s.secrets[secretName]
	if !ok {
		return nil, nil, "not-found"
	}
	num, ok := sec.ResolveVersion(versionID)
	if !ok {
		return nil, nil, "not-found"
	}
	v := sec.Versions[num]
	return versionProto(secretName, v), v, ""
}

// AccessVersion returns the payload data of an ENABLED version.
func (s *State) AccessVersion(name string) (*Version, string, string) {
	secretName, versionID, ok := ParseVersionName(name)
	if !ok {
		return nil, "", "invalid-name"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	sec, ok := s.secrets[secretName]
	if !ok {
		return nil, "", "not-found"
	}
	num, ok := sec.ResolveVersion(versionID)
	if !ok {
		return nil, "", "not-found"
	}
	v := sec.Versions[num]
	if v.State == smpb.SecretVersion_DESTROYED {
		return nil, "", "destroyed"
	}
	if v.State != smpb.SecretVersion_ENABLED {
		return nil, "", "disabled"
	}
	return v, secretName, ""
}

// SetVersionState transitions a version to ENABLED or DISABLED, or DESTROYED
// when destroy is true.
func (s *State) SetVersionState(name string, target smpb.SecretVersion_State, destroy bool) (*smpb.SecretVersion, string) {
	secretName, versionID, ok := ParseVersionName(name)
	if !ok {
		return nil, "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sec, ok := s.secrets[secretName]
	if !ok {
		return nil, "not-found"
	}
	num, ok := sec.ResolveVersion(versionID)
	if !ok {
		return nil, "not-found"
	}
	v := sec.Versions[num]
	if v.State == smpb.SecretVersion_DESTROYED {
		return nil, "destroyed"
	}
	if destroy {
		v.State = smpb.SecretVersion_DESTROYED
		t := now()
		v.DestroyTime = &t
		v.Data = nil
	} else {
		v.State = target
	}
	sec.Revision++
	return versionProto(secretName, v), ""
}

func parseStateFilter(filter string) (smpb.SecretVersion_State, string) {
	if filter == "" {
		return smpb.SecretVersion_STATE_UNSPECIFIED, ""
	}
	const prefix = "state="
	if !strings.HasPrefix(filter, prefix) {
		return 0, "invalid-filter"
	}
	switch strings.TrimPrefix(filter, prefix) {
	case "ENABLED":
		return smpb.SecretVersion_ENABLED, ""
	case "DISABLED":
		return smpb.SecretVersion_DISABLED, ""
	case "DESTROYED":
		return smpb.SecretVersion_DESTROYED, ""
	default:
		return 0, "invalid-filter"
	}
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

func cloneTopics(in []*smpb.Topic) []*smpb.Topic {
	if in == nil {
		return nil
	}
	out := make([]*smpb.Topic, len(in))
	for i, t := range in {
		out[i] = &smpb.Topic{Name: t.Name}
	}
	return out
}

func cloneRotation(in *smpb.Rotation) *smpb.Rotation {
	if in == nil {
		return nil
	}
	cp := proto.Clone(in).(*smpb.Rotation)
	cp.ManagedRotationStatus = nil // output-only
	return cp
}
