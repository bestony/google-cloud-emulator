// Package iam provides the shared IAM policy store used by products that
// expose getIamPolicy/setIamPolicy/testIamPermissions.
//
// TestIamPermissions uses a deliberately simple heuristic (documented as an
// emulator approximation): members listed in a binding whose role contains
// "owner" or "editor" may perform every requested permission; bindings whose
// role contains "viewer" or "reader" may perform read-only permissions
// (method names ending in ".get" or ".list"). The caller matches a member
// string of "user:<email>", "allUsers", "allAuthenticatedUsers" or
// "domain:<domain>" derived from the caller email.
package iam

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
)

// Binding is one role-to-members grant.
type Binding struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

// Policy is a resource's IAM policy.
type Policy struct {
	Version  int32     `json:"version"`
	Bindings []Binding `json:"bindings,omitempty"`
	Etag     string    `json:"etag,omitempty"`
}

// Store keeps policies keyed by resource name (e.g. a secret resource name or
// a managed zone resource path).
type Store struct {
	mu       sync.RWMutex
	policies map[string]*Policy
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{policies: make(map[string]*Policy)}
}

// Get returns a copy of the policy for resource, or an empty policy.
func (s *Store) Get(resource string) Policy {
	s.mu.RLock()
	p, ok := s.policies[resource]
	s.mu.RUnlock()
	if !ok || p == nil {
		return Policy{Version: 1}
	}
	cp := *p
	cp.Bindings = append([]Binding(nil), p.Bindings...)
	for i := range cp.Bindings {
		cp.Bindings[i].Members = append([]string(nil), p.Bindings[i].Members...)
	}
	return cp
}

// Set stores policy for resource and refreshes its etag.
func (s *Store) Set(resource string, p Policy) Policy {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := Policy{
		Version:  p.Version,
		Bindings: append([]Binding(nil), p.Bindings...),
	}
	if cp.Version == 0 {
		cp.Version = 1
	}
	for i := range cp.Bindings {
		cp.Bindings[i].Members = append([]string(nil), p.Bindings[i].Members...)
		sort.Strings(cp.Bindings[i].Members)
	}
	sort.Slice(cp.Bindings, func(i, j int) bool { return cp.Bindings[i].Role < cp.Bindings[j].Role })
	cp.Etag = etagOf(resource, cp)
	s.policies[resource] = &cp
	return cp
}

// Test returns the subset of permissions the caller is allowed to perform on
// resource, using the heuristic described in the package doc.
func (s *Store) Test(resource string, permissions []string, caller string) []string {
	p := s.Get(resource)
	if len(p.Bindings) == 0 {
		return nil
	}
	members := callerMembers(caller)
	var allowed []string
	for _, perm := range permissions {
		if policyAllows(p, perm, members) {
			allowed = append(allowed, perm)
		}
	}
	return allowed
}

func callerMembers(email string) []string {
	if email == "" {
		email = "dev@localhost"
	}
	members := []string{"user:" + email, "allUsers", "allAuthenticatedUsers"}
	if i := strings.LastIndex(email, "@"); i >= 0 {
		members = append(members, "domain:"+email[i+1:])
	}
	return members
}

func policyAllows(p Policy, permission string, members []string) bool {
	perm := strings.ToLower(permission)
	readOnly := strings.HasSuffix(perm, ".get") || strings.HasSuffix(perm, ".list") ||
		strings.Contains(perm, "resourcemanager.projects.get")
	for _, b := range p.Bindings {
		role := strings.ToLower(b.Role)
		if !memberMatches(b.Members, members) {
			continue
		}
		switch {
		case strings.Contains(role, "owner"), strings.Contains(role, "editor"),
			strings.Contains(role, "admin"):
			return true
		case readOnly && (strings.Contains(role, "viewer") || strings.Contains(role, "reader")):
			return true
		}
	}
	return false
}

func memberMatches(bindingMembers, callerMembers []string) bool {
	set := make(map[string]bool, len(bindingMembers))
	for _, m := range bindingMembers {
		set[m] = true
	}
	for _, m := range callerMembers {
		if set[m] {
			return true
		}
	}
	return false
}

func etagOf(resource string, p Policy) string {
	h := sha1.New()
	h.Write([]byte(resource))
	for _, b := range p.Bindings {
		h.Write([]byte{0})
		h.Write([]byte(b.Role))
		for _, m := range b.Members {
			h.Write([]byte{0})
			h.Write([]byte(m))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
