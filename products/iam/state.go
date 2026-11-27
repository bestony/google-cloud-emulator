// Package iam implements the google.iam.v1.IAMPolicy service: the generic
// get/set/test IAM policy API over arbitrary resource names. Policies live
// in the shared core/iam store, so they are visible to product-level
// getIamPolicy handlers and vice versa.
//
// IAM conditions and audit configs are not modeled by core/iam and return
// UNIMPLEMENTED rather than being silently dropped.
package iam

import (
	coreiam "github.com/bestony/google-cloud-emulator/core/iam"
)

// State wraps the shared core IAM policy store.
type State struct {
	store *coreiam.Store
}

func NewState(store *coreiam.Store) *State {
	return &State{store: store}
}

// Store exposes the underlying store for tests and wiring.
func (s *State) Store() *coreiam.Store { return s.store }
