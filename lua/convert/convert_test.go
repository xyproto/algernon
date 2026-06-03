package convert

import (
	"bytes"
	"testing"
	"time"

	lua "github.com/xyproto/gopher-lua"
)

func TestStrings2table(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	table := Strings2table(L, []string{"alpha", "beta", "gamma"})
	if table.Len() != 3 {
		t.Fatalf("expected 3 elements, got %d", table.Len())
	}
	if table.RawGetInt(1).String() != "alpha" {
		t.Errorf("element 1 = %q, want %q", table.RawGetInt(1).String(), "alpha")
	}
	if table.RawGetInt(3).String() != "gamma" {
		t.Errorf("element 3 = %q, want %q", table.RawGetInt(3).String(), "gamma")
	}

	// Empty slice
	empty := Strings2table(L, nil)
	if empty.Len() != 0 {
		t.Errorf("expected 0 elements for nil slice, got %d", empty.Len())
	}
}

func TestMap2table(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	m := map[string]string{"name": "Alice", "role": "admin"}
	table := Map2table(L, m)

	val := L.GetField(table, "name")
	if val.String() != "Alice" {
		t.Errorf("name = %q, want %q", val.String(), "Alice")
	}
	val = L.GetField(table, "role")
	if val.String() != "admin" {
		t.Errorf("role = %q, want %q", val.String(), "admin")
	}
}

func TestTable2maps(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// String keys and string values
	table := L.NewTable()
	L.RawSet(table, lua.LString("color"), lua.LString("blue"))
	L.RawSet(table, lua.LString("size"), lua.LString("large"))

	mapSS, mapSI, mapIS, mapII := Table2maps(table)
	if len(mapSS) != 2 {
		t.Fatalf("expected 2 string-string entries, got %d", len(mapSS))
	}
	if mapSS["color"] != "blue" {
		t.Errorf("color = %q, want %q", mapSS["color"], "blue")
	}
	if len(mapSI) != 0 || len(mapIS) != 0 || len(mapII) != 0 {
		t.Error("expected other maps to be empty")
	}

	// Integer keys and string values (Lua array style)
	arr := L.NewTable()
	arr.Append(lua.LString("first"))
	arr.Append(lua.LString("second"))

	_, _, mapIS2, _ := Table2maps(arr)
	if len(mapIS2) != 2 {
		t.Fatalf("expected 2 int-string entries, got %d", len(mapIS2))
	}
	if mapIS2[1] != "first" {
		t.Errorf("index 1 = %q, want %q", mapIS2[1], "first")
	}
}

func TestTable2map(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// String-String map
	table := L.NewTable()
	L.RawSet(table, lua.LString("x"), lua.LString("hello"))

	result, mixed := Table2map(table, false)
	if mixed {
		t.Error("expected non-mixed result")
	}
	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", result)
	}
	if m["x"] != "hello" {
		t.Errorf("x = %q, want %q", m["x"], "hello")
	}

	// Empty table returns nil
	empty := L.NewTable()
	result, _ = Table2map(empty, false)
	if result != nil {
		t.Errorf("expected nil for empty table, got %v", result)
	}
}

func TestLValueWrapperScan(t *testing.T) {
	var w LValueWrapper

	// nil
	if err := w.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if w.LValue != lua.LNil {
		t.Errorf("Scan(nil) = %v, want LNil", w.LValue)
	}

	// string
	if err := w.Scan("hello"); err != nil {
		t.Fatal(err)
	}
	if w.LValue.String() != "hello" {
		t.Errorf("Scan(string) = %q, want %q", w.LValue.String(), "hello")
	}

	// int64
	if err := w.Scan(int64(42)); err != nil {
		t.Fatal(err)
	}
	if w.LValue.(lua.LNumber) != lua.LNumber(42) {
		t.Errorf("Scan(int64) = %v, want 42", w.LValue)
	}

	// float64
	if err := w.Scan(3.14); err != nil {
		t.Fatal(err)
	}
	if float64(w.LValue.(lua.LNumber)) != 3.14 {
		t.Errorf("Scan(float64) = %v, want 3.14", w.LValue)
	}

	// []byte
	if err := w.Scan([]byte("bytes")); err != nil {
		t.Fatal(err)
	}
	if w.LValue.String() != "bytes" {
		t.Errorf("Scan([]byte) = %q, want %q", w.LValue.String(), "bytes")
	}

	// time.Time
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := w.Scan(ts); err != nil {
		t.Fatal(err)
	}
	if float64(w.LValue.(lua.LNumber)) != float64(ts.Unix()) {
		t.Errorf("Scan(time) = %v, want %v", w.LValue, ts.Unix())
	}

	// Unsupported type
	if err := w.Scan(struct{}{}); err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestLValueWrappersUnwrapAndInterfaces(t *testing.T) {
	wrappers := LValueWrappers{
		{lua.LString("a")},
		{lua.LNumber(1)},
		{lua.LNil},
	}

	unwrapped := wrappers.Unwrap()
	if len(unwrapped) != 3 {
		t.Fatalf("Unwrap length = %d, want 3", len(unwrapped))
	}
	if unwrapped[0].String() != "a" {
		t.Errorf("unwrap[0] = %q, want %q", unwrapped[0].String(), "a")
	}

	ifaces := wrappers.Interfaces()
	if len(ifaces) != 3 {
		t.Fatalf("Interfaces length = %d, want 3", len(ifaces))
	}
}

func TestExtractParams(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// Push a table with mixed types onto the stack
	table := L.NewTable()
	table.Append(lua.LString("hello"))
	table.Append(lua.LNumber(42))
	table.Append(lua.LBool(true))
	L.Push(table)

	params := ExtractParams(L, 1)
	if len(params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(params))
	}
	if params[0] != "hello" {
		t.Errorf("param[0] = %v, want %q", params[0], "hello")
	}
	if params[1] != float64(42) {
		t.Errorf("param[1] = %v, want 42", params[1])
	}
	if params[2] != true {
		t.Errorf("param[2] = %v, want true", params[2])
	}

	// nil table returns empty slice
	L.Push(lua.LNil)
	params = ExtractParams(L, 2)
	if len(params) != 0 {
		t.Errorf("expected 0 params for nil, got %d", len(params))
	}
}

func TestTable2JSON(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	table := L.NewTable()
	L.RawSet(table, lua.LString("name"), lua.LString("Algernon"))
	L.RawSet(table, lua.LString("version"), lua.LNumber(2))

	jsonStr, err := Table2JSON(L, table)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(jsonStr), []byte(`"name":"Algernon"`)) {
		t.Errorf("JSON should contain name field, got %s", jsonStr)
	}
	if !bytes.Contains([]byte(jsonStr), []byte(`"version":2`)) {
		t.Errorf("JSON should contain version field, got %s", jsonStr)
	}
}

func TestJSON2table(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	table := JSON2table(L, `{"greeting":"hello","count":3,"active":true}`)

	val := L.GetField(table, "greeting")
	if val.String() != "hello" {
		t.Errorf("greeting = %q, want %q", val.String(), "hello")
	}
	val = L.GetField(table, "count")
	if float64(val.(lua.LNumber)) != 3 {
		t.Errorf("count = %v, want 3", val)
	}
	val = L.GetField(table, "active")
	if val != lua.LTrue {
		t.Errorf("active = %v, want true", val)
	}

	// Invalid JSON returns empty table
	empty := JSON2table(L, "not json at all")
	if empty.Len() != 0 {
		t.Errorf("invalid JSON should return empty table, got len %d", empty.Len())
	}
}

func TestPprintToWriter(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// String value
	var buf bytes.Buffer
	PprintToWriter(&buf, lua.LString("hello"))
	if buf.String() != "hello" {
		t.Errorf("pprint string = %q, want %q", buf.String(), "hello")
	}

	// Number value
	buf.Reset()
	PprintToWriter(&buf, lua.LNumber(42))
	if buf.String() != "42" {
		t.Errorf("pprint number = %q, want %q", buf.String(), "42")
	}

	// Empty table
	buf.Reset()
	PprintToWriter(&buf, L.NewTable())
	if buf.String() != "{}" {
		t.Errorf("pprint empty table = %q, want %q", buf.String(), "{}")
	}

	// Array-style table
	buf.Reset()
	arr := L.NewTable()
	arr.Append(lua.LString("x"))
	arr.Append(lua.LString("y"))
	PprintToWriter(&buf, arr)
	got := buf.String()
	if got != `{"x", "y"}` {
		t.Errorf("pprint array = %q, want %q", got, `{"x", "y"}`)
	}
}

func TestArguments2buffer(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.Push(lua.LString("hello"))
	L.Push(lua.LString("world"))

	buf := Arguments2buffer(L, true)
	if buf.String() != "hello world\n" {
		t.Errorf("Arguments2buffer = %q, want %q", buf.String(), "hello world\n")
	}

	// Without newline
	L2 := lua.NewState()
	defer L2.Close()
	L2.Push(lua.LNumber(42))
	buf = Arguments2buffer(L2, false)
	if buf.String() != "42" {
		t.Errorf("Arguments2buffer no newline = %q, want %q", buf.String(), "42")
	}
}
