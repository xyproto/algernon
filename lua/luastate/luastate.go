// Package luastate wraps lua/pool with a Borrow/Return API and scoped helpers.
// The underlying pool package stays exported for external users of algernon as a library.
package luastate

import (
	"github.com/xyproto/algernon/lua/pool"
	lua "github.com/xyproto/gopher-lua"
)

// Pool wraps lua/pool.LStatePool with a Borrow/Return discipline
type Pool struct {
	lp *pool.LStatePool
}

// New returns a new Pool
func New() *Pool {
	return &Pool{lp: pool.New()}
}

// Underlying returns the wrapped *pool.LStatePool, for interop with not-yet-migrated code
func (p *Pool) Underlying() *pool.LStatePool {
	return p.lp
}

// Borrow takes a Lua state out of the pool, or creates a new one. Pair with Return.
func (p *Pool) Borrow() *lua.LState {
	return p.lp.Get()
}

// Return delivers back a borrowed Lua state
func (p *Pool) Return(L *lua.LState) {
	p.lp.Put(L)
}

// New returns a fresh Lua state that is not pooled; the caller is responsible for closing it
func (p *Pool) New() *lua.LState {
	return p.lp.New()
}

// With borrows a Lua state, runs fn, and returns the state to the pool even on panic
func (p *Pool) With(fn func(L *lua.LState) error) (err error) {
	L := p.lp.Get()
	defer p.lp.Put(L)
	return fn(L)
}

// WithNew runs fn with a fresh (non-pooled) Lua state and closes it afterwards, even on panic
func (p *Pool) WithNew(fn func(L *lua.LState) error) (err error) {
	L := p.lp.New()
	defer L.Close()
	return fn(L)
}

// Shutdown closes all pooled Lua states
func (p *Pool) Shutdown() {
	p.lp.Shutdown()
}
