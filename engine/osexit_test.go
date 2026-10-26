package engine

import (
	"strings"
	"testing"

	lua "github.com/xyproto/gopher-lua"
)

func TestDisableOSExitRaisesError(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	disableOSExit(L)

	err := L.DoString(`os.exit(1)`)
	if err == nil || !strings.Contains(err.Error(), "os.exit is not allowed") {
		t.Fatalf("expected os.exit to raise a Lua error, got %v", err)
	}
	if L.IsClosed() {
		t.Fatal("the Lua state was closed by os.exit")
	}
	if err := L.DoString(`x = 1`); err != nil {
		t.Fatal(err)
	}
}
