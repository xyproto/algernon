package main

import (
	"net/http"
	"strings"

	"github.com/xyproto/permissions2"
	"github.com/xyproto/simpleredis"
	"github.com/yuin/gopher-lua"
)

// Identifier for the Hash class in Lua
const lHashClass = "HASH"

// Get the first argument, "self", and cast it from userdata to a hash map.
func checkHash(L *lua.LState) *simpleredis.HashMap {
	ud := L.CheckUserData(1)
	if set, ok := ud.Value.(*simpleredis.HashMap); ok {
		return set
	}
	L.ArgError(1, "hash map expected")
	return nil
}

// Create a new hash map.
// id is the name of the hash map.
// dbindex is the Redis database index (typically 0).
func newHashMap(L *lua.LState, pool *simpleredis.ConnectionPool, id string, dbindex int) (*lua.LUserData, error) {
	// Create a new simpleredis hash map
	hash := simpleredis.NewHashMap(pool, id)
	hash.SelectDatabase(dbindex)
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = hash
	L.SetMetatable(ud, L.GetTypeMetatable(lHashClass))
	return ud, nil
}

// String representation
// Returns the entire hash map as a comma separated string
// tostring(hash) -> string
func hashToString(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	all, err := hash.GetAll()
	if err != nil {
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}
	L.Push(lua.LString(strings.Join(all, ", ")))
	return 1 // Number of returned values
}

// For a given element id (for instance a user id), set a key (for instance "password") and a value.
// Returns true if it worked out.
// hash:set(string, string, string) -> bool
func hashSet(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.ToString(2)
	key := L.ToString(3)
	value := L.ToString(4)
	L.Push(lua.LBool(nil == hash.Set(elementid, key, value)))
	return 1 // Number of returned values
}

// For a given element id (for instance a user id), and a key (for instance "password"), return a value.
// Returns a value only if they key was found and if there were no errors.
// hash:get(string, string) -> string
func hashGet(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.ToString(2)
	key := L.ToString(3)
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
	elementid := L.ToString(2)
	key := L.ToString(3)
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
	elementid := L.ToString(2)
	b, err := hash.Exists(elementid)
	if err != nil {
		b = false
	}
	L.Push(lua.LBool(b))
	return 1 // Number of returned values
}

// Get all members of the set
func hashGetAll(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	all, err := hash.GetAll()
	if err != nil {
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}
	L.Push(strings2table(L, all))
	return 1 // Number of returned values
}

// Remove a key for an entry in a hash map (for instance the email field for a user)
// Returns true if it worked out
// hash:delkey(string, string) -> bool
func hashDelKey(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.ToString(2)
	key := L.ToString(3)
	L.Push(lua.LBool(nil == hash.DelKey(elementid, key)))
	return 1 // Number of returned values
}

// Remove an element (for instance a user)
// Returns true if it worked out
// hash:del(string) -> bool
func hashDel(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	elementid := L.ToString(2)
	L.Push(lua.LBool(nil == hash.Del(elementid)))
	return 1 // Number of returned values
}

// Remove the hash map itself. Returns true if it worked out.
// hash:remove() -> bool
func hashRemove(L *lua.LState) int {
	hash := checkHash(L) // arg 1
	L.Push(lua.LBool(nil == hash.Remove()))
	return 1 // Number of returned values
}

// The hash map methods that are to be registered
var hashMethods = map[string]lua.LGFunction{
	"__tostring": hashToString,
	"set":        hashSet,
	"get":        hashGet,
	"has":        hashHas,
	"exists":     hashExists,
	"getall":     hashGetAll,
	"delkey":     hashDelKey,
	"del":        hashDel,
	"remove":     hashRemove,
}

// Make functions related to HTTP requests and responses available to Lua scripts
func exportHash(w http.ResponseWriter, req *http.Request, L *lua.LState, userstate *permissions.UserState) {
	pool := userstate.Pool()
	dbindex := userstate.DatabaseIndex()

	// Register the hash map class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lHashClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, hashMethods)

	// The constructor for new hash maps takes a name and an optional redis db index
	L.SetGlobal("NewHashMap", L.NewFunction(func(L *lua.LState) int {
		name := L.ToString(1)

		// Check if the optional argument is given
		localDBIndex := dbindex
		if L.GetTop() == 2 {
			localDBIndex = L.ToInt(2)
		}

		// Create a new hash map in Lua
		userdata, err := newHashMap(L, pool, name, localDBIndex)
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
