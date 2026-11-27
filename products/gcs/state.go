// Package gcs implements the Cloud Storage JSON API control plane:
// bucket CRUD, list, and IAM policy verbs under /storage/v1/.
// Object data plane requests return honest UNIMPLEMENTED errors.
package gcs

import (
	"sync"

	"github.com/bestony/google-cloud-emulator/core/iam"
)

// State holds buckets keyed by name (bucket names are globally unique in
// GCS). Each bucket is stored as raw JSON so client-set fields (labels,
// cors, lifecycle, website, ...) round-trip without being modeled.
type State struct {
	mu      sync.RWMutex
	buckets map[string]map[string]any
	iam     *iam.Store
}

// NewState returns a State backed by the shared IAM store.
func NewState(iamStore *iam.Store) *State {
	return &State{buckets: map[string]map[string]any{}, iam: iamStore}
}

// IAM returns the shared IAM store; bucket policies are keyed by
// "projects/_/buckets/{bucket}", matching the GCS IAM resource naming.
func (s *State) IAM() *iam.Store { return s.iam }

func iamKey(bucket string) string { return "projects/_/buckets/" + bucket }

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
}
