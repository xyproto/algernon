package engine

import (
	lua "github.com/xyproto/gopher-lua"
)

// handlerPool is a bounded pool of Lua states used to serve handle()
// requests. Each state holds its own copy of every handle() function in its
// Lua registry, keyed by path. A buffered channel provides Get/Put with
// natural backpressure: if all states are busy, Get blocks until one is
// returned.
type handlerPool struct {
	ch     chan *lua.LState
	states []*lua.LState // kept for Close()
}

// newHandlerPool creates a pool of the given size. The caller must add
// states with Add() until the pool is full.
func newHandlerPool(size int) *handlerPool {
	if size < 1 {
		size = 1
	}
	return &handlerPool{
		ch:     make(chan *lua.LState, size),
		states: make([]*lua.LState, 0, size),
	}
}

// Add enqueues a fully prepared state into the pool. Called during pool
// construction only, so no lock is needed for the states slice.
func (p *handlerPool) Add(L *lua.LState) {
	p.states = append(p.states, L)
	p.ch <- L
}

// Get borrows a state from the pool, blocking if all are in use
func (p *handlerPool) Get() *lua.LState {
	return <-p.ch
}

// Put returns a state to the pool
func (p *handlerPool) Put(L *lua.LState) {
	p.ch <- L
}

// Close shuts down all states. In-flight handlers holding a state are
// not tracked; close is best-effort for server shutdown.
func (p *handlerPool) Close() {
	for _, L := range p.states {
		L.Close()
	}
}
