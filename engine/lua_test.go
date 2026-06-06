package engine

import (
	"testing"

	lua "github.com/xyproto/gopher-lua"
)

// Covers the helper used by the recursive table conversion in issue #119.
func TestIsArrayLikeTable(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	t.Run("list of numbers", func(t *testing.T) {
		tbl := L.NewTable()
		tbl.Append(lua.LNumber(1))
		tbl.Append(lua.LNumber(2))
		tbl.Append(lua.LNumber(3))
		if !isArrayLikeTable(tbl) {
			t.Error("expected {1,2,3} to be array-like")
		}
	})

	t.Run("list of nested tables (issue #119)", func(t *testing.T) {
		outer := L.NewTable()
		for i := 1; i <= 3; i++ {
			inner := L.NewTable()
			inner.Append(lua.LNumber(i))
			outer.Append(inner)
		}
		if !isArrayLikeTable(outer) {
			t.Error("expected {{1},{2},{3}} to be array-like")
		}
	})

	t.Run("string-keyed map", func(t *testing.T) {
		tbl := L.NewTable()
		tbl.RawSetString("a", lua.LNumber(1))
		tbl.RawSetString("b", lua.LNumber(2))
		if isArrayLikeTable(tbl) {
			t.Error("string-keyed table must not be array-like")
		}
	})

	t.Run("empty table", func(t *testing.T) {
		tbl := L.NewTable()
		if isArrayLikeTable(tbl) {
			t.Error("empty table must not be array-like")
		}
	})

	t.Run("sparse table", func(t *testing.T) {
		tbl := L.NewTable()
		tbl.RawSetInt(1, lua.LNumber(10))
		tbl.RawSetInt(3, lua.LNumber(30))
		if isArrayLikeTable(tbl) && tbl.RawGetInt(2) == lua.LNil {
			t.Error("sparse table with a nil hole must not be array-like")
		}
	})
}
