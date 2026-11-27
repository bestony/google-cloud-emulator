// Package kms emulates google.cloud.kms.v1 KeyManagementService over gRPC
// and REST (via grpc-gateway). State is in-memory only.
//
// M1 scope: key rings, crypto keys and crypto key versions control plane.
// Crypto operations (Encrypt/Decrypt/Asymmetric*/Mac*/Decapsulate/
// GenerateRandomBytes), import jobs and retired resources return
// UNIMPLEMENTED.
package kms

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bestony/google-cloud-emulator/core/lro"
	"github.com/bestony/google-cloud-emulator/core/pagination"
	kmspb "github.com/bestony/google-cloud-emulator/gen/google/cloud/kms/v1"
)

var (
	keyRingNameRe   = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+/keyRings/[^/]+$`)
	cryptoKeyNameRe = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+/keyRings/[^/]+/cryptoKeys/[^/]+$`)
	versionNameRe   = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+/keyRings/[^/]+/cryptoKeys/[^/]+/cryptoKeyVersions/[^/]+$`)
	keyRingIDRe     = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,63}$`)
	cryptoKeyIDRe   = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,63}$`)
)

// CryptoKeyEntry is a crypto key plus its versions.
type CryptoKeyEntry struct {
	Proto           *kmspb.CryptoKey
	Versions        map[string]*kmspb.CryptoKeyVersion // version id (numeric string) -> version
	Material        map[string]*keyMaterial            // version id -> generated key material
	NextVersion     int64
	TrustedWrapping bool
}

// State holds all KMS resources in memory.
type State struct {
	mu         sync.RWMutex
	keyRings   map[string]*kmspb.KeyRing
	cryptoKeys map[string]*CryptoKeyEntry
	lro        *lro.Store
}

// NewState returns an empty State backed by the shared LRO store.
func NewState(lroStore *lro.Store) *State {
	return &State{
		keyRings:   make(map[string]*kmspb.KeyRing),
		cryptoKeys: make(map[string]*CryptoKeyEntry),
		lro:        lroStore,
	}
}

// LROStore exposes the shared LRO store.
func (s *State) LROStore() *lro.Store { return s.lro }

func now() *timestamppb.Timestamp { return timestamppb.Now() }

// ValidKeyRingName reports whether name is a well-formed key ring resource name.
func ValidKeyRingName(name string) bool { return keyRingNameRe.MatchString(name) }

// ValidCryptoKeyName reports whether name is a well-formed crypto key resource name.
func ValidCryptoKeyName(name string) bool { return cryptoKeyNameRe.MatchString(name) }

// ValidVersionName reports whether name is a well-formed crypto key version resource name.
func ValidVersionName(name string) bool { return versionNameRe.MatchString(name) }

// SplitVersionName returns (cryptoKeyName, versionID).
func SplitVersionName(name string) (string, string, bool) {
	if !versionNameRe.MatchString(name) {
		return "", "", false
	}
	idx := strings.LastIndex(name, "/cryptoKeyVersions/")
	return name[:idx], name[idx+len("/cryptoKeyVersions/"):], true
}

// --- Key rings ---

// CreateKeyRing creates a key ring; code is "" or "already-exists"/"invalid-name".
func (s *State) CreateKeyRing(parent, keyRingID string, kr *kmspb.KeyRing) (*kmspb.KeyRing, string) {
	if !strings.HasPrefix(parent, "projects/") || !strings.Contains(parent, "/locations/") {
		return nil, "invalid-parent"
	}
	if !keyRingIDRe.MatchString(keyRingID) {
		return nil, "invalid-name"
	}
	name := strings.TrimSuffix(parent, "/") + "/keyRings/" + keyRingID
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keyRings[name]; ok {
		return nil, "already-exists"
	}
	out := &kmspb.KeyRing{Name: name, CreateTime: now()}
	s.keyRings[name] = out
	return proto.Clone(out).(*kmspb.KeyRing), ""
}

// GetKeyRing returns a copy or nil.
func (s *State) GetKeyRing(name string) *kmspb.KeyRing {
	s.mu.RLock()
	defer s.mu.RUnlock()
	kr, ok := s.keyRings[name]
	if !ok {
		return nil
	}
	return proto.Clone(kr).(*kmspb.KeyRing)
}

// ListKeyRings returns a sorted page of key rings under parent.
func (s *State) ListKeyRings(parent string, pageSize int, pageToken string) ([]*kmspb.KeyRing, string, int, string) {
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var names []string
	prefix := strings.TrimSuffix(parent, "/") + "/keyRings/"
	for name := range s.keyRings {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	total := len(names)
	if offset >= total {
		return []*kmspb.KeyRing{}, "", total, ""
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	out := make([]*kmspb.KeyRing, 0, end-offset)
	for _, n := range names[offset:end] {
		out = append(out, proto.Clone(s.keyRings[n]).(*kmspb.KeyRing))
	}
	next := ""
	if end < total {
		next = pagination.Encode(end)
	}
	return out, next, total, ""
}

// --- Crypto keys ---

// defaultAlgorithm picks the version algorithm for a purpose/protection level.
func defaultAlgorithm(purpose kmspb.CryptoKey_CryptoKeyPurpose, protection kmspb.ProtectionLevel) kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm {
	if protection == kmspb.ProtectionLevel_HSM || protection == kmspb.ProtectionLevel_HSM_SINGLE_TENANT {
		// HSM keys still use the software algorithm identifiers for symmetric.
	}
	switch purpose {
	case kmspb.CryptoKey_ASYMMETRIC_SIGN:
		return kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256
	case kmspb.CryptoKey_ASYMMETRIC_DECRYPT:
		return kmspb.CryptoKeyVersion_RSA_DECRYPT_OAEP_2048_SHA256
	case kmspb.CryptoKey_MAC:
		return kmspb.CryptoKeyVersion_HMAC_SHA256
	default:
		return kmspb.CryptoKeyVersion_GOOGLE_SYMMETRIC_ENCRYPTION
	}
}

// CreateCryptoKey creates a crypto key with an initial version unless skipped.
func (s *State) CreateCryptoKey(parent, cryptoKeyID string, ck *kmspb.CryptoKey, skipInitial bool, trustedWrapping bool) (*kmspb.CryptoKey, string) {
	if !keyRingNameRe.MatchString(parent) {
		return nil, "invalid-parent"
	}
	if !cryptoKeyIDRe.MatchString(cryptoKeyID) {
		return nil, "invalid-name"
	}
	name := strings.TrimSuffix(parent, "/") + "/cryptoKeys/" + cryptoKeyID
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keyRings[parent]; !ok {
		return nil, "not-found"
	}
	if _, ok := s.cryptoKeys[name]; ok {
		return nil, "already-exists"
	}
	src := proto.Clone(ck).(*kmspb.CryptoKey)
	entry := &CryptoKeyEntry{
		Proto: &kmspb.CryptoKey{
			Name:                     name,
			Purpose:                  src.GetPurpose(),
			CreateTime:               now(),
			NextRotationTime:         cloneTimestamp(src.NextRotationTime),
			VersionTemplate:          cloneVersionTemplate(src.VersionTemplate),
			Labels:                   cloneStringMap(src.Labels),
			ImportOnly:               src.GetImportOnly(),
			RotationSchedule:         src.RotationSchedule,
			DestroyScheduledDuration: cloneDuration(src.DestroyScheduledDuration),
			CryptoKeyBackend:         src.GetCryptoKeyBackend(),
		},
		Versions:        make(map[string]*kmspb.CryptoKeyVersion),
		NextVersion:     1,
		TrustedWrapping: trustedWrapping,
	}
	if !skipInitial {
		s.addVersionLocked(entry, "1")
		entry.NextVersion = 2
		entry.Proto.Primary = proto.Clone(entry.Versions["1"]).(*kmspb.CryptoKeyVersion)
	}
	s.cryptoKeys[name] = entry
	return proto.Clone(entry.Proto).(*kmspb.CryptoKey), ""
}

// addVersionLocked appends a new ENABLED version; caller holds the lock.
func (s *State) addVersionLocked(entry *CryptoKeyEntry, versionID string) {
	protection := kmspb.ProtectionLevel_SOFTWARE
	algorithm := defaultAlgorithm(entry.Proto.GetPurpose(), protection)
	if vt := entry.Proto.GetVersionTemplate(); vt != nil {
		if vt.GetProtectionLevel() != kmspb.ProtectionLevel_PROTECTION_LEVEL_UNSPECIFIED {
			protection = vt.GetProtectionLevel()
		}
		if vt.GetAlgorithm() != kmspb.CryptoKeyVersion_CRYPTO_KEY_VERSION_ALGORITHM_UNSPECIFIED {
			algorithm = vt.GetAlgorithm()
		}
	}
	v := &kmspb.CryptoKeyVersion{
		Name:                   entry.Proto.GetName() + "/cryptoKeyVersions/" + versionID,
		State:                  kmspb.CryptoKeyVersion_ENABLED,
		ProtectionLevel:        protection,
		Algorithm:              algorithm,
		CreateTime:             now(),
		GenerateTime:           now(),
		TrustedWrappingEnabled: entry.TrustedWrapping,
	}
	entry.Versions[versionID] = v
}

// GetCryptoKey returns a copy or nil.
func (s *State) GetCryptoKey(name string) *kmspb.CryptoKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.cryptoKeys[name]
	if !ok {
		return nil
	}
	return proto.Clone(entry.Proto).(*kmspb.CryptoKey)
}

// ListCryptoKeys returns a sorted page under parent (a key ring name).
func (s *State) ListCryptoKeys(parent string, pageSize int, pageToken string) ([]*kmspb.CryptoKey, string, int, string) {
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var names []string
	prefix := strings.TrimSuffix(parent, "/") + "/cryptoKeys/"
	for name := range s.cryptoKeys {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	total := len(names)
	if offset >= total {
		return []*kmspb.CryptoKey{}, "", total, ""
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	out := make([]*kmspb.CryptoKey, 0, end-offset)
	for _, n := range names[offset:end] {
		out = append(out, proto.Clone(s.cryptoKeys[n].Proto).(*kmspb.CryptoKey))
	}
	next := ""
	if end < total {
		next = pagination.Encode(end)
	}
	return out, next, total, ""
}

// cryptoKeyMutableFields are the UpdateCryptoKey mask paths the emulator supports.
var cryptoKeyMutableFields = map[string]bool{
	"labels":                     true,
	"next_rotation_time":         true,
	"rotation_period":            true,
	"rotation_schedule":          true,
	"version_template":           true,
	"destroy_scheduled_duration": true,
}

// UpdateCryptoKey applies whitelisted mask paths; code "" or error string.
func (s *State) UpdateCryptoKey(patch *kmspb.CryptoKey, maskPaths []string) (*kmspb.CryptoKey, string) {
	name := patch.GetName()
	if !ValidCryptoKeyName(name) {
		return nil, "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cryptoKeys[name]
	if !ok {
		return nil, "not-found"
	}
	pc := proto.Clone(patch).(*kmspb.CryptoKey)
	for _, p := range maskPaths {
		switch p {
		case "labels":
			entry.Proto.Labels = pc.Labels
		case "next_rotation_time":
			entry.Proto.NextRotationTime = pc.NextRotationTime
		case "version_template":
			entry.Proto.VersionTemplate = pc.VersionTemplate
		case "destroy_scheduled_duration":
			entry.Proto.DestroyScheduledDuration = pc.DestroyScheduledDuration
		case "rotation_period", "rotation_schedule":
			entry.Proto.RotationSchedule = pc.RotationSchedule
		default:
			return nil, "invalid-mask:" + p
		}
	}
	return proto.Clone(entry.Proto).(*kmspb.CryptoKey), ""
}

// DeleteCryptoKeyEntry removes a key and its versions; returns the deleted key copy.
func (s *State) DeleteCryptoKeyEntry(name string) *kmspb.CryptoKey {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cryptoKeys[name]
	if !ok {
		return nil
	}
	out := proto.Clone(entry.Proto).(*kmspb.CryptoKey)
	delete(s.cryptoKeys, name)
	return out
}

// --- Crypto key versions ---

// GetCryptoKeyVersion returns a copy of a version or nil.
func (s *State) GetCryptoKeyVersion(name string) *kmspb.CryptoKeyVersion {
	keyName, id, ok := SplitVersionName(name)
	if !ok {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.cryptoKeys[keyName]
	if !ok {
		return nil
	}
	v, ok := entry.Versions[id]
	if !ok {
		return nil
	}
	return proto.Clone(v).(*kmspb.CryptoKeyVersion)
}

// ListCryptoKeyVersions returns a sorted page of versions under parent (crypto key name).
func (s *State) ListCryptoKeyVersions(parent string, pageSize int, pageToken string) ([]*kmspb.CryptoKeyVersion, string, int, string) {
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.cryptoKeys[parent]
	if !ok {
		return []*kmspb.CryptoKeyVersion{}, "", 0, ""
	}
	var ids []string
	for id := range entry.Versions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return versionLess(ids[i], ids[j]) })
	total := len(ids)
	if offset >= total {
		return []*kmspb.CryptoKeyVersion{}, "", total, ""
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	out := make([]*kmspb.CryptoKeyVersion, 0, end-offset)
	for _, id := range ids[offset:end] {
		out = append(out, proto.Clone(entry.Versions[id]).(*kmspb.CryptoKeyVersion))
	}
	next := ""
	if end < total {
		next = pagination.Encode(end)
	}
	return out, next, total, ""
}

func versionLess(a, b string) bool {
	ai, bi := int64(0), int64(0)
	fmt.Sscanf(a, "%d", &ai)
	fmt.Sscanf(b, "%d", &bi)
	if ai != bi {
		return ai < bi
	}
	return a < b
}

// CreateCryptoKeyVersion creates a new version for asymmetric-purpose keys.
func (s *State) CreateCryptoKeyVersion(parent string, ver *kmspb.CryptoKeyVersion) (*kmspb.CryptoKeyVersion, string) {
	if !ValidCryptoKeyName(parent) {
		return nil, "invalid-parent"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cryptoKeys[parent]
	if !ok {
		return nil, "not-found"
	}
	purpose := entry.Proto.GetPurpose()
	switch purpose {
	case kmspb.CryptoKey_ASYMMETRIC_SIGN, kmspb.CryptoKey_ASYMMETRIC_DECRYPT, kmspb.CryptoKey_MAC:
	default:
		return nil, "invalid-purpose"
	}
	id := fmt.Sprintf("%d", entry.NextVersion)
	entry.NextVersion++
	s.addVersionLocked(entry, id)
	created := entry.Versions[id]
	if ver.GetProtectionLevel() != kmspb.ProtectionLevel_PROTECTION_LEVEL_UNSPECIFIED {
		created.ProtectionLevel = ver.GetProtectionLevel()
	}
	if ver.GetAlgorithm() != kmspb.CryptoKeyVersion_CRYPTO_KEY_VERSION_ALGORITHM_UNSPECIFIED {
		created.Algorithm = ver.GetAlgorithm()
	}
	return proto.Clone(created).(*kmspb.CryptoKeyVersion), ""
}

// UpdateCryptoKeyPrimaryVersion points the key's primary at an existing version.
func (s *State) UpdateCryptoKeyPrimaryVersion(name, versionID string) (*kmspb.CryptoKey, string) {
	if !ValidCryptoKeyName(name) {
		return nil, "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cryptoKeys[name]
	if !ok {
		return nil, "not-found"
	}
	v, ok := entry.Versions[versionID]
	if !ok {
		return nil, "version-not-found"
	}
	if v.GetState() == kmspb.CryptoKeyVersion_DESTROYED || v.GetState() == kmspb.CryptoKeyVersion_DESTROY_SCHEDULED {
		return nil, "version-destroyed"
	}
	entry.Proto.Primary = proto.Clone(v).(*kmspb.CryptoKeyVersion)
	return proto.Clone(entry.Proto).(*kmspb.CryptoKey), ""
}

// UpdateCryptoKeyVersionState changes a version's state (ENABLED/DISABLED).
func (s *State) UpdateCryptoKeyVersionState(name string, state kmspb.CryptoKeyVersion_CryptoKeyVersionState) (*kmspb.CryptoKeyVersion, string) {
	keyName, id, ok := SplitVersionName(name)
	if !ok {
		return nil, "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cryptoKeys[keyName]
	if !ok {
		return nil, "not-found"
	}
	v, ok := entry.Versions[id]
	if !ok {
		return nil, "not-found"
	}
	if v.GetState() == kmspb.CryptoKeyVersion_DESTROYED || v.GetState() == kmspb.CryptoKeyVersion_DESTROY_SCHEDULED {
		return nil, "version-destroyed"
	}
	if state != kmspb.CryptoKeyVersion_ENABLED && state != kmspb.CryptoKeyVersion_DISABLED {
		return nil, "invalid-state"
	}
	v.State = state
	return proto.Clone(v).(*kmspb.CryptoKeyVersion), ""
}

// DestroyCryptoKeyVersion schedules a version for destruction.
func (s *State) DestroyCryptoKeyVersion(name string) (*kmspb.CryptoKeyVersion, string) {
	keyName, id, ok := SplitVersionName(name)
	if !ok {
		return nil, "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cryptoKeys[keyName]
	if !ok {
		return nil, "not-found"
	}
	v, ok := entry.Versions[id]
	if !ok {
		return nil, "not-found"
	}
	if v.GetState() == kmspb.CryptoKeyVersion_DESTROY_SCHEDULED || v.GetState() == kmspb.CryptoKeyVersion_DESTROYED {
		return nil, "already-destroyed"
	}
	d := entry.Proto.GetDestroyScheduledDuration()
	if d == nil {
		d = defaultDestroyDuration
	}
	v.State = kmspb.CryptoKeyVersion_DESTROY_SCHEDULED
	v.DestroyTime = timestamppb.New(time.Now().Add(d.AsDuration()).UTC())
	return proto.Clone(v).(*kmspb.CryptoKeyVersion), ""
}

// RestoreCryptoKeyVersion restores a DESTROY_SCHEDULED version to ENABLED.
func (s *State) RestoreCryptoKeyVersion(name string) (*kmspb.CryptoKeyVersion, string) {
	keyName, id, ok := SplitVersionName(name)
	if !ok {
		return nil, "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cryptoKeys[keyName]
	if !ok {
		return nil, "not-found"
	}
	v, ok := entry.Versions[id]
	if !ok {
		return nil, "not-found"
	}
	if v.GetState() != kmspb.CryptoKeyVersion_DESTROY_SCHEDULED {
		return nil, "not-restore-scheduled"
	}
	v.State = kmspb.CryptoKeyVersion_ENABLED
	v.DestroyTime = nil
	return proto.Clone(v).(*kmspb.CryptoKeyVersion), ""
}

var defaultDestroyDuration = durationpb.New(24 * time.Hour)

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

func cloneTimestamp(in *timestamppb.Timestamp) *timestamppb.Timestamp {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*timestamppb.Timestamp)
}

func cloneDuration(in *durationpb.Duration) *durationpb.Duration {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*durationpb.Duration)
}

func cloneVersionTemplate(in *kmspb.CryptoKeyVersionTemplate) *kmspb.CryptoKeyVersionTemplate {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*kmspb.CryptoKeyVersionTemplate)
}
