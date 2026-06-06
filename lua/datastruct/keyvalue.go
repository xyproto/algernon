package datastruct

import (
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/xyproto/gluamapper"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/pinterface/v2"
)

// Identifier for the Set class in Lua
const lKeyValueClass = "KEYVALUE"

// Prefix used to mark KeyValue entries that hold a JSON-encoded Lua table.
// Starts with a NUL byte so plain strings cannot collide. See issue #113.
const kvTableMarker = "\x00alg-json:"

// kvGluamapperOption keeps the original key names when converting Lua tables.
var kvGluamapperOption = gluamapper.Option{
	NameFunc: func(s string) string { return s },
}

// anyToLua converts a JSON-decoded Go value back into a Lua value.
func anyToLua(L *lua.LState, v any) lua.LValue {
	switch val := v.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(val)
	case float64:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	case []any:
		t := L.NewTable()
		for i, item := range val {
			t.RawSetInt(i+1, anyToLua(L, item))
		}
		return t
	case map[string]any:
		t := L.NewTable()
		for k, item := range val {
			t.RawSetString(k, anyToLua(L, item))
		}
		return t
	}
	return lua.LNil
}

// Get the first argument, "self", and cast it from userdata to a key/value
func checkKeyValue(L *lua.LState) pinterface.IKeyValue {
	ud := L.CheckUserData(1)
	if kv, ok := ud.Value.(pinterface.IKeyValue); ok {
		return kv
	}
	L.ArgError(1, "keyvalue expected")
	return nil
}

// Create a new KeyValue collection.
// id is the name of the KeyValue collection.
// dbindex is the Redis database index (typically 0).
func newKeyValue(L *lua.LState, creator pinterface.ICreator, id string) (*lua.LUserData, error) {
	// Create a new key/value
	kv, err := creator.NewKeyValue(id)
	if err != nil {
		return nil, err
	}
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = kv
	L.SetMetatable(ud, L.GetTypeMetatable(lKeyValueClass))
	return ud, nil
}

// String representation
// Returns the name of the KeyValue collection
// tostring(kv) -> string
func kvToString(L *lua.LState) int {
	L.Push(lua.LString("keyvalue"))
	return 1 // Number of returned values
}

// Set a key and value. Returns true if successful.
// Tables are JSON-encoded and reconstructed on get, see issue #113.
// kv:set(string, string|table) -> bool
func kvSet(L *lua.LState) int {
	kv := checkKeyValue(L) // arg 1
	key := L.CheckString(2)
	var value string
	if tbl, ok := L.Get(3).(*lua.LTable); ok {
		data, err := json.Marshal(gluamapper.ToGoValue(tbl, kvGluamapperOption))
		if err != nil {
			logrus.Error(err)
			L.Push(lua.LBool(false))
			return 1
		}
		value = kvTableMarker + string(data)
	} else {
		value = L.ToString(3)
	}
	L.Push(lua.LBool(nil == kv.Set(key, value)))
	return 1 // Number of returned values
}

// Takes a key, returns a value. May return an empty string.
// If the stored value was a table, returns a table instead of a string.
// kv:get(string) -> string|table
func kvGet(L *lua.LState) int {
	kv := checkKeyValue(L) // arg 1
	key := L.CheckString(2)
	retval, err := kv.Get(key)
	if err != nil {
		retval = ""
	}
	if strings.HasPrefix(retval, kvTableMarker) {
		var decoded any
		if err := json.Unmarshal([]byte(retval[len(kvTableMarker):]), &decoded); err == nil {
			L.Push(anyToLua(L, decoded))
			return 1
		}
	}
	L.Push(lua.LString(retval))
	return 1 // Number of returned values
}

// Takes a key, returns the value+1.
// Creates a key/value and returns "1" if it did not already exist.
// May return an empty string.
// kv:inc(string) -> string
func kvInc(L *lua.LState) int {
	kv := checkKeyValue(L) // arg 1
	key := L.CheckString(2)
	increased, err := kv.Inc(key)
	if err != nil {
		logrus.Error(err.Error())
		L.Push(lua.LString("0"))
		return 1
	}
	L.Push(lua.LString(increased))
	return 1 // Number of returned values
}

// Remove a key. Returns true if successful.
// kv:del(string) -> bool
func kvDel(L *lua.LState) int {
	kv := checkKeyValue(L) // arg 1
	value := L.CheckString(2)
	L.Push(lua.LBool(nil == kv.Del(value)))
	return 1 // Number of returned values
}

// Remove the keyvalue itself. Returns true if successful.
// kv:remove() -> bool
func kvRemove(L *lua.LState) int {
	kv := checkKeyValue(L) // arg 1
	L.Push(lua.LBool(nil == kv.Remove()))
	return 1 // Number of returned values
}

// Clear the keyvalue. Returns true if successful.
// kv:clear() -> bool
func kvClear(L *lua.LState) int {
	kv := checkKeyValue(L) // arg 1
	L.Push(lua.LBool(nil == kv.Clear()))
	return 1 // Number of returned values
}

// The keyvalue methods that are to be registered
var kvMethods = map[string]lua.LGFunction{
	"__tostring": kvToString,
	"set":        kvSet,
	"get":        kvGet,
	"inc":        kvInc,
	"del":        kvDel,
	"remove":     kvRemove,
	"clear":      kvClear,
}

// LoadKeyValue makes functions related to HTTP requests and responses available to Lua scripts
func LoadKeyValue(L *lua.LState, creator pinterface.ICreator) {
	// Register the KeyValue class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lKeyValueClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, kvMethods)

	// The constructor for new KeyValues takes a name and an optional redis db index
	L.SetGlobal("KeyValue", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)

		// Check if the optional argument is given
		if L.GetTop() == 2 {
			localDBIndex := L.ToInt(2)

			// Set the DB index, if possible
			switch rh := creator.(type) {
			case pinterface.IRedisCreator:
				rh.SelectDatabase(localDBIndex)
			}
		}

		// Create a new keyvalue in Lua
		userdata, err := newKeyValue(L, creator, name)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			L.Push(lua.LNumber(1))
			return 3 // Number of returned values
		}

		// Return the keyvalue object
		L.Push(userdata)
		return 1 // Number of returned values
	}))
}
