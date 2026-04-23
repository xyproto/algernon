package luastate

import (
	"errors"
	"sync"
	"testing"

	lua "github.com/xyproto/gopher-lua"
)

func TestBorrowReturn(t *testing.T) {
	p := New()
	defer p.Shutdown()

	L := p.Borrow()
	if L == nil {
		t.Fatal("Borrow returned nil")
	}
	p.Return(L)

	// After return, a subsequent Borrow should hand us the same state back
	L2 := p.Borrow()
	if L2 != L {
		t.Errorf("expected the same Lua state to be reused, got a fresh one")
	}
	p.Return(L2)
}

func TestWithRunsFn(t *testing.T) {
	p := New()
	defer p.Shutdown()

	var ran bool
	err := p.With(func(L *lua.LState) error {
		ran = true
		return L.DoString(`x = 1 + 2`)
	})
	if err != nil {
		t.Fatalf("With returned error: %v", err)
	}
	if !ran {
		t.Fatal("fn was not invoked")
	}
}

func TestWithPropagatesError(t *testing.T) {
	p := New()
	defer p.Shutdown()

	sentinel := errors.New("boom")
	err := p.With(func(L *lua.LState) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}

	// The state should still be returned to the pool
	L := p.Borrow()
	if L == nil {
		t.Fatal("expected the state to have been returned to the pool")
	}
	p.Return(L)
}

func TestWithNewIsolated(t *testing.T) {
	p := New()
	defer p.Shutdown()

	err := p.WithNew(func(L *lua.LState) error {
		return L.DoString(`y = 42`)
	})
	if err != nil {
		t.Fatalf("WithNew returned error: %v", err)
	}
}

func TestConcurrentWith(t *testing.T) {
	p := New()
	defer p.Shutdown()

	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = p.With(func(L *lua.LState) error {
				return L.DoString(`z = 1`)
			})
		}()
	}
	wg.Wait()
}
