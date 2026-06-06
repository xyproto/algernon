package datastruct

import (
	"encoding/json"
	"testing"

	"github.com/xyproto/gluamapper"
	lua "github.com/xyproto/gopher-lua"
)

// Round-trip a typical Lua table through the helpers used by kv:set/kv:get for issue #113.
func TestLuaTableJSONRoundTrip(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	src := L.NewTable()
	src.RawSetString("a", lua.LNumber(100))
	src.RawSetString("b", lua.LString("bee"))
	inner := L.NewTable()
	inner.Append(lua.LNumber(1))
	inner.Append(lua.LNumber(2))
	inner.Append(lua.LNumber(3))
	src.RawSetString("c", inner)

	encoded := gluamapper.ToGoValue(src, kvGluamapperOption)
	roundtripped := anyToLua(L, jsonRoundTrip(t, encoded))

	tbl, ok := roundtripped.(*lua.LTable)
	if !ok {
		t.Fatalf("expected table, got %T", roundtripped)
	}
	if got := tbl.RawGetString("a"); got != lua.LNumber(100) {
		t.Errorf("a = %v, want 100", got)
	}
	if got := tbl.RawGetString("b"); got != lua.LString("bee") {
		t.Errorf("b = %v, want bee", got)
	}
	c, ok := tbl.RawGetString("c").(*lua.LTable)
	if !ok {
		t.Fatalf("c is not a table: %T", tbl.RawGetString("c"))
	}
	for i, want := range []lua.LNumber{1, 2, 3} {
		if got := c.RawGetInt(i + 1); got != want {
			t.Errorf("c[%d] = %v, want %v", i+1, got, want)
		}
	}
}

// jsonRoundTrip exercises the same encode/decode path that kv:set/kv:get use.
func jsonRoundTrip(t *testing.T, v any) any {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}
