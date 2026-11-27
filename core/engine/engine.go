// Package engine is the shared state registry: products register their
// in-memory state under a key at startup and handlers look it up from there.
package engine

import (
	"fmt"
	"sync"
)

// Engine holds per-product state values.
type Engine struct {
	mu     sync.RWMutex
	states map[string]any
}

// New returns an empty Engine.
func New() *Engine {
	return &Engine{states: make(map[string]any)}
}

// Register stores v under key. It panics on duplicate keys so wiring mistakes
// surface at startup.
func (e *Engine) Register(key string, v any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, dup := e.states[key]; dup {
		panic(fmt.Sprintf("engine: duplicate state key %q", key))
	}
	e.states[key] = v
}

// Get returns the value registered under key.
func (e *Engine) Get(key string) (any, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	v, ok := e.states[key]
	return v, ok
}

// MustGet returns the value registered under key or panics.
func (e *Engine) MustGet(key string) any {
	v, ok := e.Get(key)
	if !ok {
		panic(fmt.Sprintf("engine: no state registered for %q", key))
	}
	return v
}
