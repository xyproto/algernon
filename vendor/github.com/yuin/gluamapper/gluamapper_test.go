package gluamapper

import (
	"github.com/yuin/gopher-lua"
	"path/filepath"
	"runtime"
	"testing"
)

func errorIfNotEqual(t *testing.T, v1, v2 interface{}) {
	if v1 != v2 {
		_, file, line, _ := runtime.Caller(1)
		t.Errorf("%v line %v: '%v' expected, but got '%v'", filepath.Base(file), line, v1, v2)
	}
}

type testRole struct {
	Name string
}

type testPerson struct {
	Name      string
	Age       int
	WorkPlace string
	Role      []*testRole
}

type testStruct struct {
	Nil    interface{}
	Bool   bool
	String string
	Number int `gluamapper:"number_value"`
	Func   interface{}
}

func TestMap(t *testing.T) {
	L := lua.NewState()
	if err := L.DoString(`
    person = {
      name = "Michel",
      age  = "31", -- weakly input
      work_place = "San Jose",
      role = {
        {
          name = "Administrator"
        },
        {
          name = "Operator"
        }
      }
    }
	`); err != nil {
		t.Error(err)
	}
	var person testPerson
	if err := Map(L.GetGlobal("person").(*lua.LTable), &person); err != nil {
		t.Error(err)
	}
	errorIfNotEqual(t, "Michel", person.Name)
	errorIfNotEqual(t, 31, person.Age)
	errorIfNotEqual(t, "San Jose", person.WorkPlace)
	errorIfNotEqual(t, 2, len(person.Role))
	errorIfNotEqual(t, "Administrator", person.Role[0].Name)
	errorIfNotEqual(t, "Operator", person.Role[1].Name)
}

func TestTypes(t *testing.T) {
	L := lua.NewState()
	if err := L.DoString(`
    tbl = {
      ["Nil"] = nil,
      ["Bool"] = true,
      ["String"] = "string",
      ["Number_value"] = 10,
      ["Func"] = function() end
    }
	`); err != nil {
		t.Error(err)
	}
	var stct testStruct

	if err := NewMapper(Option{NameFunc: Id}).Map(L.GetGlobal("tbl").(*lua.LTable), &stct); err != nil {
		t.Error(err)
	}
	errorIfNotEqual(t, nil, stct.Nil)
	errorIfNotEqual(t, true, stct.Bool)
	errorIfNotEqual(t, "string", stct.String)
	errorIfNotEqual(t, 10, stct.Number)
}

func TestNameFunc(t *testing.T) {
	L := lua.NewState()
	if err := L.DoString(`
    person = {
      Name = "Michel",
      Age  = "31", -- weekly input
      WorkPlace = "San Jose",
      Role = {
        {
          Name = "Administrator"
        },
        {
          Name = "Operator"
        }
      }
    }
	`); err != nil {
		t.Error(err)
	}
	var person testPerson
	mapper := NewMapper(Option{NameFunc: Id})
	if err := mapper.Map(L.GetGlobal("person").(*lua.LTable), &person); err != nil {
		t.Error(err)
	}
	errorIfNotEqual(t, "Michel", person.Name)
	errorIfNotEqual(t, 31, person.Age)
	errorIfNotEqual(t, "San Jose", person.WorkPlace)
	errorIfNotEqual(t, 2, len(person.Role))
	errorIfNotEqual(t, "Administrator", person.Role[0].Name)
	errorIfNotEqual(t, "Operator", person.Role[1].Name)
}

func TestError(t *testing.T) {
	L := lua.NewState()
	tbl := L.NewTable()
	L.SetField(tbl, "key", lua.LString("value"))
	err := Map(tbl, 1)
	if err.Error() != "result must be a pointer" {
		t.Error("invalid error message")
	}

	tbl = L.NewTable()
	tbl.Append(lua.LNumber(1))
	var person testPerson
	err = Map(tbl, &person)
	if err.Error() != "arguments #1 must be a table, but got an array" {
		t.Error("invalid error message")
	}
}
