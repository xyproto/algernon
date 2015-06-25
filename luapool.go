package main

import (
	"github.com/yuin/gopher-lua"
	"sync"
)

// The LState pool pattern, as recommended by the author of gopher-lua:
// https://github.com/yuin/gopher-lua#the-lstate-pool-pattern

type lStatePool struct {
	m     sync.Mutex
	saved []*lua.LState
}

func (pl *lStatePool) Get() *lua.LState {
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

func (pl *lStatePool) New() *lua.LState {
	L := lua.NewState()
	// setting the L up here.
	// load scripts, set global variables, share channels, etc...
	return L
}

func (pl *lStatePool) Put(L *lua.LState) {
	pl.m.Lock()
	defer pl.m.Unlock()
	pl.saved = append(pl.saved, L)
}

func (pl *lStatePool) Shutdown() {
	// The following line causes a race condition with the
	// graceful shutdown package at server shutdown:
	//for _, L := range pl.saved {
	//	L.Close()
	//}
}
