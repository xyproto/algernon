package datastruct

import (
	"strings"

	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/gopher-lua"
	"github.com/xyproto/pinterface"
)

// Identifier for the Hash class in Lua
const lHashClass = "HASH"

// Get the first argument, "self", and cast it from userdata to a hash map.
func checkHash(L *lua.LState) pinterface.IHashMap {
	ud := L.CheckUserData(1)
	if hash, ok := ud.Value.(pinterface.IHashMap); ok {
		return hash
	}
	L.ArgError(1, "hash map expected")
	return nil
}

// Create a new hash map.
// id is the name of the hash map.
// dbindex is the Redis database index (typically 0).
func newHashMap(L *lua.LState, creator pinterface.ICreator, id string) (*lua.LUserData, error) {
	// Create a new hash map
	hash, err := creator.NewHashMap(id)
	if err != nil {
		return nil, err
	}
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = hash
	L.SetMetatable(ud, L.GetTypeMetatable(lHashClass))
	return ud, nil
}

// String representation
// Returns all keys in the hash map as a comma separated string
// tostring(hash) -> string
func hashToString(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	all, err := hash.All()
	if err != nil {
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}
	L.Push(lua.LString(strings.Join(all, ", ")))
	return 1 // Number of returned values
}

// For a given element id (for instance a user id), set a key (for instance "password") and a value.
// Returns true if successful.
// hash:set(string, string, string) -> bool
func hashSet(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.CheckString(2)
	key := L.CheckString(3)
	value := L.ToString(4)
	L.Push(lua.LBool(nil == hash.Set(elementid, key, value)))
	return 1 // Number of returned values
}

// For a given element id (for instance a user id), and a key (for instance "password"), return a value.
// Returns a value only if they key was found and if there were no errors.
// hash:get(string, string) -> string
func hashGet(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.CheckString(2)
	key := L.CheckString(3)
	retval, err := hash.Get(elementid, key)
	if err != nil {
		retval = ""
	}
	L.Push(lua.LString(retval))
	return 1 // Number of returned values
}

// For a given element id (for instance a user id), and a key (for instance "password"), check if it exists in the hash map.
// Returns true only if it exists and there were no errors.
// hash:has(string, string) -> bool
func hashHas(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.CheckString(2)
	key := L.CheckString(3)
	b, err := hash.Has(elementid, key)
	if err != nil {
		b = false
	}
	L.Push(lua.LBool(b))
	return 1 // Number of returned values
}

// For a given element id (for instance a user id), check if it exists in the hash map.
// Returns true only if it exists and there were no errors.
// hash:exists(string) -> bool
func hashExists(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.CheckString(2)
	b, err := hash.Exists(elementid)
	if err != nil {
		b = false
	}
	L.Push(lua.LBool(b))
	return 1 // Number of returned values
}

// Get all keys of the hash map
// hash::getall() -> table
func hashAll(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	all, err := hash.All()
	if err != nil {
		// Return an empty table
		L.Push(L.NewTable())
		return 1 // Number of returned values
	}
	L.Push(convert.Strings2table(L, all))
	return 1 // Number of returned values
}

// Get all subkeys of the hash map
// hash::keys() -> table
func hashKeys(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.CheckString(2)
	keys, err := hash.Keys(elementid)
	if err != nil {
		// Return an empty table
		L.Push(L.NewTable())
		return 1 // Number of returned values
	}
	L.Push(convert.Strings2table(L, keys))
	return 1 // Number of returned values
}

// Remove a key for an entry in a hash map (for instance the email field for a user)
// Returns true if successful
// hash:delkey(string, string) -> bool
func hashDelKey(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.CheckString(2)
	key := L.CheckString(3)
	L.Push(lua.LBool(nil == hash.DelKey(elementid, key)))
	return 1 // Number of returned values
}

// Remove an element (for instance a user)
// Returns true if successful
// hash:del(string) -> bool
func hashDel(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.CheckString(2)
	L.Push(lua.LBool(nil == hash.Del(elementid)))
	return 1 // Number of returned values
}

// Remove the hash map itself. Returns true if successful.
// hash:remove() -> bool
func hashRemove(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	L.Push(lua.LBool(nil == hash.Remove()))
	return 1 // Number of returned values
}

// Clear the hash map. Returns true if successful.
// hash:clear() -> bool
func hashClear(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	L.Push(lua.LBool(nil == hash.Clear()))
	return 1 // Number of returned values
}

// The hash map methods that are to be registered
var hashMethods = map[string]lua.LGFunction{
	"__tostring": hashToString,
	"set":        hashSet,
	"get":        hashGet,
	"has":        hashHas,
	"exists":     hashExists,
	"getall":     hashAll,
	"keys":       hashKeys,
	"delkey":     hashDelKey,
	"del":        hashDel,
	"remove":     hashRemove,
	"clear":      hashClear,
}

// LoadHash makes functions related to HTTP requests and responses available to Lua scripts
func LoadHash(L *lua.LState, creator pinterface.ICreator) {

	// Register the hash map class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lHashClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, hashMethods)

	// The constructor for new hash maps takes a name and an optional redis db index
	L.SetGlobal("HashMap", L.NewFunction(func(L *lua.LState) int {
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

		// Create a new hash map in Lua
		userdata, err := newHashMap(L, creator, name)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			L.Push(lua.LNumber(1))
			return 3 // Number of returned values
		}

		// Return the hash map object
		L.Push(userdata)
		return 1 // Number of returned values
	}))

}
