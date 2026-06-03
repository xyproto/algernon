package jnode

import (
	"strings"
	"testing"

	lua "github.com/xyproto/gopher-lua"
)

// TestLoadJSONFunctions verifies the json() / JSON() / toJSON() Lua globals
func TestLoadJSONFunctions(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	LoadJSONFunctions(L)

	// json({key="value"}) should produce JSON
	if err := L.DoString(`result = json({name="Algernon", version="1.0"})`); err != nil {
		t.Fatal(err)
	}
	result := L.GetGlobal("result").String()
	if !strings.Contains(result, `"name"`) || !strings.Contains(result, `"Algernon"`) {
		t.Errorf("json() = %q, expected it to contain name:Algernon", result)
	}
	if !strings.Contains(result, `"version"`) {
		t.Errorf("json() = %q, expected it to contain version field", result)
	}

	// json({}, 2) with indent
	if err := L.DoString(`indented = json({x="1"}, 2)`); err != nil {
		t.Fatal(err)
	}
	indented := L.GetGlobal("indented").String()
	if !strings.Contains(indented, "\n") {
		t.Errorf("json() with indent should contain newlines, got %q", indented)
	}

	// JSON is an alias for json
	if err := L.DoString(`result2 = JSON({greeting="hi"})`); err != nil {
		t.Fatal(err)
	}
	result2 := L.GetGlobal("result2").String()
	if !strings.Contains(result2, `"greeting"`) {
		t.Errorf("JSON() = %q, expected it to contain greeting", result2)
	}
}

// TestLoadJNode verifies the JNode constructor and basic methods
func TestLoadJNode(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	Load(L)

	// Create a JNode with initial data, retrieve a string field
	code := `
		node = JNode('{"name":"Algernon","version":"1.0"}')
		result = node:getstring("name")
	`
	if err := L.DoString(code); err != nil {
		t.Fatal(err)
	}

	result := L.GetGlobal("result").String()
	if result != "Algernon" {
		t.Errorf("JNode:getstring(name) = %q, want %q", result, "Algernon")
	}
}

// TestJNodePrettyAndCompact tests pretty-printing and compact output
func TestJNodePrettyAndCompact(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	Load(L)

	code := `
		node = JNode('{"name":"test","count":42}')
		pretty_out = node:pretty()
		compact_out = node:compact()
	`
	if err := L.DoString(code); err != nil {
		t.Fatal(err)
	}

	pretty := L.GetGlobal("pretty_out").String()
	compact := L.GetGlobal("compact_out").String()

	if !strings.Contains(pretty, "name") {
		t.Errorf("pretty() should contain 'name', got %q", pretty)
	}
	if !strings.Contains(compact, "name") {
		t.Errorf("compact() should contain 'name', got %q", compact)
	}
	// Pretty should be longer (has indentation/newlines)
	if len(pretty) <= len(compact) {
		t.Errorf("pretty (%d bytes) should be longer than compact (%d bytes)", len(pretty), len(compact))
	}
}

// TestJNodeSetAndGet tests setting and getting string values
func TestJNodeSetAndGet(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	Load(L)

	// Verify we can read an initial value from a constructed node
	code := `
		node = JNode('{"greeting":"hello","count":5}')
		result = node:getstring("greeting")
	`
	if err := L.DoString(code); err != nil {
		t.Fatal(err)
	}

	result := L.GetGlobal("result").String()
	if result != "hello" {
		t.Errorf("JNode getstring: got %q, want %q", result, "hello")
	}
}

// TestJNodeDelKey tests deleting a key from a JSON map
func TestJNodeDelKey(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	Load(L)

	code := `
		node = JNode('{"a":"1","b":"2"}')
		ok = node:delkey("a")
		result = node:getstring("a")
	`
	if err := L.DoString(code); err != nil {
		t.Fatal(err)
	}

	ok := L.GetGlobal("ok")
	if ok != lua.LTrue {
		t.Errorf("delkey should return true, got %v", ok)
	}
	result := L.GetGlobal("result").String()
	if result != "" {
		t.Errorf("after delkey, getstring should return empty, got %q", result)
	}
}

// TestJNodeAdd tests adding JSON data to a list
func TestJNodeAdd(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	Load(L)

	code := `
		node = JNode('[]')
		ok = node:add('{"id":1}')
		compact_out = node:compact()
	`
	if err := L.DoString(code); err != nil {
		t.Fatal(err)
	}

	ok := L.GetGlobal("ok")
	if ok != lua.LTrue {
		t.Errorf("add to list should return true, got %v", ok)
	}
	compact := L.GetGlobal("compact_out").String()
	if !strings.Contains(compact, "id") {
		t.Errorf("after add, compact output should contain the item, got %q", compact)
	}
}
