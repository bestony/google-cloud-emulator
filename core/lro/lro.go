// Package lro is a generic store for google.longrunning.Operation resources.
// Products that model long-running operations register an operation here,
// complete it when the work finishes, and expose Get/List through their own
// surface (the shape of the operation resource differs per product, so each
// product serializes these records itself).
package lro

import (
	"fmt"
	"sort"
	"sync"
	"time"

	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
)

// Operation is one tracked unit of work.
type Operation struct {
	// Name is the fully-qualified operation resource name.
	Name string
	// Kind labels the operation for product-specific metadata.
	Kind string
	// Done reports completion.
	Done bool
	// Err is set when the operation failed.
	Err *gcperr.Code
	// Message is the failure message when Err is set.
	Message string
	// Result is the product-specific success payload.
	Result any
	// CreateTime is when the operation started.
	CreateTime time.Time
	// UpdateTime is when it last changed state.
	UpdateTime time.Time
}

// Store keeps operations by name.
type Store struct {
	mu   sync.RWMutex
	ops  map[string]*Operation
	next int64
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{ops: make(map[string]*Operation)}
}

// Start creates an operation named "<namePrefix>/<id>" and returns it. The
// caller must eventually call Complete or Fail.
func (s *Store) Start(namePrefix, kind string) *Operation {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	op := &Operation{
		Name:       fmt.Sprintf("%s/%d", namePrefix, s.next),
		Kind:       kind,
		CreateTime: time.Now().UTC(),
		UpdateTime: time.Now().UTC(),
	}
	s.ops[op.Name] = op
	return op
}

// Get returns the operation with the given name.
func (s *Store) Get(name string) (*Operation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	op, ok := s.ops[name]
	return op, ok
}

// Complete marks op done with result.
func (s *Store) Complete(op *Operation, result any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	op.Done = true
	op.Result = result
	op.UpdateTime = time.Now().UTC()
}

// Fail marks op done with an error.
func (s *Store) Fail(op *Operation, code gcperr.Code, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	op.Done = true
	op.Err = &code
	op.Message = message
	op.UpdateTime = time.Now().UTC()
}

// List returns operations whose name has prefix, newest first.
func (s *Store) List(prefix string) []*Operation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Operation
	for _, op := range s.ops {
		if len(op.Name) >= len(prefix) && op.Name[:len(prefix)] == prefix {
			out = append(out, op)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name })
	return out
}
