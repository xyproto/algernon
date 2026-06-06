// Package pool provides functions for managing a pool of Lua state structs
package pool

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/lua/teal"
	lua "github.com/xyproto/gopher-lua"
)

// The LState pool pattern, as recommended by the author of gopher-lua:
// https://github.com/xyproto/gopher-lua#the-lstate-pool-pattern

// LStatePool is a pool of Lua states, with a mutex
type LStatePool struct {
	saved      []*lua.LState
	globalsLua []byte
	m          sync.Mutex
}

// New returns a new Lua pool structure
func New() *LStatePool {
	return &LStatePool{saved: make([]*lua.LState, 0, 4)}
}

// SetGlobalsScript stores Lua code to run on every freshly-created state in
// the pool. Used to share globals.lua across the request handlers. Must be
// called before the pool is used concurrently.
func (pl *LStatePool) SetGlobalsScript(code []byte) {
	pl.globalsLua = code
}

// New returns a new Lua state, and sets the context
func (pl *LStatePool) New() *lua.LState {
	L := lua.NewState()
	ctx := context.Background()
	L.SetContext(ctx)

	// Teal
	teal.Load(L)

	// Apply globals.lua, if configured
	if len(pl.globalsLua) > 0 {
		if err := L.DoString(string(pl.globalsLua)); err != nil {
			logrus.Errorf("globals.lua: %s", err)
		}
	}

	return L
}

// Get borrows an existing Lua state, but sets a new context
func (pl *LStatePool) Get() *lua.LState {
	pl.m.Lock()
	defer pl.m.Unlock()
	n := len(pl.saved)
	if n == 0 {
		return pl.New()
	}
	x := pl.saved[n-1]
	pl.saved = pl.saved[0 : n-1]
	return x
}

// Put delivers back a borrowed Lua state
func (pl *LStatePool) Put(L *lua.LState) {
	pl.m.Lock()
	defer pl.m.Unlock()
	pl.saved = append(pl.saved, L)
}

// Shutdown can be used then the Lua pool is being shut down
func (pl *LStatePool) Shutdown() {
	// The following lines previously causesd a race condition with the
	// graceful shutdown package at server shutdown:
	for _, L := range pl.saved {
		L.Close()
	}
}
