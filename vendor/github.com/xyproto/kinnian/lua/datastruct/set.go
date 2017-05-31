package datastruct

import (
	"strings"

	"github.com/xyproto/kinnian/lua/convert"
	"github.com/xyproto/pinterface"
	"github.com/yuin/gopher-lua"
)

// Identifier for the Set class in Lua
const lSetClass = "SET"

// Get the first argument, "self", and cast it from userdata to a set.
func checkSet(L *lua.LState) pinterface.ISet {
	ud := L.CheckUserData(1)
	if set, ok := ud.Value.(pinterface.ISet); ok {
		return set
	}
	L.ArgError(1, "set expected")
	return nil
}

// Create a new set.
// id is the name of the set.
// dbindex is the Redis database index (typically 0).
func newSet(L *lua.LState, creator pinterface.ICreator, id string) (*lua.LUserData, error) {
	// Create a new set
	set, err := creator.NewSet(id)
	if err != nil {
		return nil, err
	}
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = set
	L.SetMetatable(ud, L.GetTypeMetatable(lSetClass))
	return ud, nil
}

// String representation
// Returns the entire set as a comma separated string
// tostring(set) -> string
func setToString(L *lua.LState) int {
	set := checkSet(L) // arg 1
	all, err := set.GetAll()
	if err != nil {
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}
	L.Push(lua.LString(strings.Join(all, ", ")))
	return 1 // Number of returned values
}

// Add an element to the set
// set:add(string)
func setAdd(L *lua.LState) int {
	set := checkSet(L) // arg 1
	value := L.ToString(2)
	set.Add(value)
	return 0 // Number of returned values
}

// Remove an element from the set
// set:del(string)
func setDel(L *lua.LState) int {
	set := checkSet(L) // arg 1
	value := L.ToString(2)
	set.Del(value)
	return 0 // Number of returned values
}

// Check if a set contains a value
// Returns true only if the value exists and there were no errors.
// set:has(string) -> bool
func setHas(L *lua.LState) int {
	set := checkSet(L) // arg 1
	value := L.ToString(2)
	b, err := set.Has(value)
	if err != nil {
		b = false
	}
	L.Push(lua.LBool(b))
	return 1 // Number of returned values
}

// Get all members of the set
// set:getall() -> table
func setGetAll(L *lua.LState) int {
	set := checkSet(L) // arg 1
	all, err := set.GetAll()
	if err != nil {
		// Return an empty table
		L.Push(L.NewTable())
		return 1 // Number of returned values
	}
	L.Push(convert.Strings2table(L, all))
	return 1 // Number of returned values
}

// Remove the set itself. Returns true if successful.
// set:remove() -> bool
func setRemove(L *lua.LState) int {
	set := checkSet(L) // arg 1
	L.Push(lua.LBool(nil == set.Remove()))
	return 1 // Number of returned values
}

// Clear the set. Returns true if successful.
// set:clear() -> bool
func setClear(L *lua.LState) int {
	set := checkSet(L) // arg 1
	L.Push(lua.LBool(nil == set.Clear()))
	return 1 // Number of returned values
}

// The set methods that are to be registered
var setMethods = map[string]lua.LGFunction{
	"__tostring": setToString,
	"add":        setAdd,
	"del":        setDel,
	"has":        setHas,
	"getall":     setGetAll,
	"remove":     setRemove,
	"clear":      setClear,
}

// LoadSet makes functions related to HTTP requests and responses available to Lua scripts
func LoadSet(L *lua.LState, creator pinterface.ICreator) {

	// Register the set class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lSetClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, setMethods)

	// The constructor for new sets takes a name and an optional redis db index
	L.SetGlobal("Set", L.NewFunction(func(L *lua.LState) int {
		name := L.ToString(1)

		// Check if the optional argument is given
		if L.GetTop() == 2 {
			localDBIndex := L.ToInt(2)

			// Set the DB index, if possible
			switch rh := creator.(type) {
			case pinterface.IRedisCreator:
				rh.SelectDatabase(localDBIndex)
			}
		}

		// Create a new set in Lua
		userdata, err := newSet(L, creator, name)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			L.Push(lua.LNumber(1))
			return 3 // Number of returned values
		}

		// Return the set object
		L.Push(userdata)
		return 1 // Number of returned values
	}))

}
