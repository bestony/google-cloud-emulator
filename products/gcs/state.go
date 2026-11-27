// Package gcs implements the Cloud Storage JSON API: bucket CRUD, IAM policy
// verbs, object metadata, object bytes, uploads, copies and composition under
// /storage/v1/ and /upload/storage/v1/.
package gcs

import (
	"fmt"
	"sync"

	"github.com/bestony/google-cloud-emulator/core/iam"
)

// objectVersion is one generation of an object. Only the newest generation is
// live; older generations are retained only when bucket versioning is enabled.
type objectVersion struct {
	metadata   map[string]any
	data       []byte
	generation int64
	live       bool
}

type objectHistory struct {
	versions []*objectVersion // newest first
}

type uploadSession struct {
	bucket   string
	metadata map[string]any
	data     []byte
}

// ObjectListEntry is one object or object generation in a list result.
type ObjectListEntry struct {
	Name     string
	Metadata map[string]any
}

// State holds buckets keyed by name (bucket names are globally unique in
// GCS). Buckets and object metadata are stored as raw JSON so client-set
// fields round-trip without being modeled.
type State struct {
	mu             sync.RWMutex
	buckets        map[string]map[string]any
	objects        map[string]map[string]*objectHistory
	sessions       map[string]*uploadSession
	nextGeneration int64
	nextSession    int64
	iam            *iam.Store
}

// NewState returns a State backed by the shared IAM store.
func NewState(iamStore *iam.Store) *State {
	return &State{
		buckets:        map[string]map[string]any{},
		objects:        map[string]map[string]*objectHistory{},
		sessions:       map[string]*uploadSession{},
		nextGeneration: 1,
		iam:            iamStore,
	}
}

// IAM returns the shared IAM store; bucket policies are keyed by
// "projects/_/buckets/{bucket}", matching the GCS IAM resource naming.
func (s *State) IAM() *iam.Store { return s.iam }

func iamKey(bucket string) string { return "projects/_/buckets/" + bucket }

func cloneValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return cloneMap(x)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = cloneValue(item)
		}
		return out
	default:
		return v
	}
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func (s *State) get(name string) (map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.buckets[name]
	return b, ok
}

func (s *State) list() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.buckets))
	for _, b := range s.buckets {
		out = append(out, b)
	}
	return out
}

func (s *State) put(name string, b map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buckets[name] = b
}

func (s *State) delete(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buckets, name)
	delete(s.objects, name)
}

// newGeneration allocates a process-unique object generation.
func (s *State) newGeneration() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	gen := s.nextGeneration
	s.nextGeneration++
	return gen
}

func versioningEnabled(bucket map[string]any) bool {
	versioning, _ := bucket["versioning"].(map[string]any)
	enabled, _ := versioning["enabled"].(bool)
	return enabled
}

// putObjectVersion inserts a new live generation. Existing generations are
// retained only when versioning is enabled on the bucket.
func (s *State) putObjectVersion(bucket, name string, metadata map[string]any, data []byte, generation int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.buckets[bucket]
	if !ok {
		return false
	}
	if s.objects[bucket] == nil {
		s.objects[bucket] = map[string]*objectHistory{}
	}
	history := s.objects[bucket][name]
	if history == nil || !versioningEnabled(b) {
		history = &objectHistory{}
		s.objects[bucket][name] = history
	}
	for _, version := range history.versions {
		version.live = false
	}
	history.versions = append([]*objectVersion{{
		metadata:   cloneMap(metadata),
		data:       cloneBytes(data),
		generation: generation,
		live:       true,
	}}, history.versions...)
	return true
}

func copyVersion(v *objectVersion) (map[string]any, []byte) {
	return cloneMap(v.metadata), cloneBytes(v.data)
}

// getObject returns object metadata and bytes. generation zero means the live
// generation; archived generations must be requested explicitly.
func (s *State) getObject(bucket, name string, generation int64) (map[string]any, []byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	history := s.objects[bucket][name]
	if history == nil || len(history.versions) == 0 {
		return nil, nil, false
	}
	if generation == 0 {
		v := history.versions[0]
		if !v.live {
			return nil, nil, false
		}
		m, d := copyVersion(v)
		return m, d, true
	}
	for _, v := range history.versions {
		if v.generation == generation {
			m, d := copyVersion(v)
			return m, d, true
		}
	}
	return nil, nil, false
}

// replaceObjectMetadata atomically replaces the metadata of one generation.
func (s *State) replaceObjectMetadata(bucket, name string, generation int64, metadata map[string]any) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	history := s.objects[bucket][name]
	if history == nil {
		return false
	}
	for _, v := range history.versions {
		if v.generation == generation {
			v.metadata = cloneMap(metadata)
			return true
		}
	}
	return false
}

// listObjects returns object metadata sorted by neither name nor generation;
// callers sort after applying list filters. Without versions only live objects
// are returned.
func (s *State) listObjects(bucket string, versions bool) []ObjectListEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []ObjectListEntry{}
	for name, history := range s.objects[bucket] {
		for _, v := range history.versions {
			if !versions && !v.live {
				continue
			}
			out = append(out, ObjectListEntry{Name: name, Metadata: cloneMap(v.metadata)})
		}
	}
	return out
}

// deleteObject removes the live generation (generation zero) or one explicit
// generation. Archived generations do not become live when the live generation
// is deleted, matching versioned-object semantics.
func (s *State) deleteObject(bucket, name string, generation int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	history := s.objects[bucket][name]
	if history == nil {
		return false
	}
	idx := -1
	if generation == 0 {
		if len(history.versions) > 0 && history.versions[0].live {
			idx = 0
		}
	} else {
		for i, v := range history.versions {
			if v.generation == generation {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return false
	}
	history.versions = append(history.versions[:idx], history.versions[idx+1:]...)
	if len(history.versions) == 0 {
		delete(s.objects[bucket], name)
	}
	return true
}

// createUploadSession stores resumable-upload parameters and returns the
// opaque upload id. The object itself is created only when the session is
// completed.
func (s *State) createUploadSession(bucket string, metadata map[string]any) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.buckets[bucket]; !ok {
		return "", false
	}
	s.nextSession++
	id := fmt.Sprintf("gcpex-upload-%d", s.nextSession)
	s.sessions[id] = &uploadSession{bucket: bucket, metadata: cloneMap(metadata)}
	return id, true
}

func (s *State) uploadSessionByID(id string) (string, map[string]any, []byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	if !ok {
		return "", nil, nil, false
	}
	return sess.bucket, cloneMap(sess.metadata), cloneBytes(sess.data), true
}

func (s *State) appendUploadSessionData(id string, data []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return false
	}
	sess.data = append(sess.data, data...)
	return true
}

func (s *State) deleteUploadSession(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}
