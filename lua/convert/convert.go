// Package convert provides functions for converting to and from Lua structures
package convert

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/gluamapper"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/jpath"
)

var (
	errToMap = errors.New("could not represent Lua structure table as a map")
)

// PprintToWriter outputs more informative information than the memory location.
// Attempt to extract and print the values of the given lua.LValue.
// Does not add a newline at the end.
func PprintToWriter(w io.Writer, value lua.LValue) {
	switch v := value.(type) {
	case *lua.LTable:
		t := (*lua.LTable)(v)
		// Even if t.Len() is 0, the table may be full of elements
		m, isAnArray, err := Table2interfaceMapGlua(t)
		if err != nil {
			//log.Info("try: for k,v in pairs(t) do pprint(k,v) end")
			// Could not convert to a map
			fmt.Fprint(w, v)
			return
		}
		if isAnArray {
			// A map which is really an array (arrays in Lua are maps)
			var buf bytes.Buffer
			buf.WriteString("{")
			// Order the map
			length := len(m)
			for i := 1; i <= length; i++ {
				// gluamapper uses float64 for all numbers?
				if val, ok := m[float64(i)]; ok {
					buf.WriteString(fmt.Sprintf("%#v", val))
					if i != length {
						// Output a comma for every element except the last one
						buf.WriteString(", ")
					}
				} else if val, ok := m[i]; ok {
					buf.WriteString(fmt.Sprintf("%#v", val))
					if i != length {
						// Output a comma for every element except the last one
						buf.WriteString(", ")
					}
				} else {
					// Unrecognized type of array or map, just return the sprintf representation
					buf.Reset()
					buf.WriteString(fmt.Sprintf("%v", m))
					buf.WriteTo(w)
					return
				}
			}
			buf.WriteString("}")
			buf.WriteTo(w)
			return
		}
		if len(m) == 0 {
			// An empty map
			fmt.Fprint(w, "{}")
			return
		}
		// Convert the map to a string
		// First extract the keys, and sort them
		var mapKeys []string
		stringMap := make(map[string]string)
		for k, v := range m {
			keyString := fmt.Sprintf("%#v", k)
			keyString = strings.TrimPrefix(keyString, "\"")
			keyString = strings.TrimSuffix(keyString, "\"")
			valueString := fmt.Sprintf("%#v", v)
			mapKeys = append(mapKeys, keyString)
			stringMap[keyString] = valueString
		}
		sort.Strings(mapKeys)

		// Then loop over the keys and build a string
		var sb strings.Builder
		sb.WriteString("{")
		for i, keyString := range mapKeys {
			if i != 0 {
				sb.WriteString(", ")
			}
			valueString := stringMap[keyString]
			sb.WriteString(keyString + "=" + valueString)
		}
		sb.WriteString("}")

		// Then replace "[]interface {}" with nothing and output the string
		s := strings.ReplaceAll(sb.String(), "[]interface {}", "")
		fmt.Fprint(w, s)
	case *lua.LFunction:
		if v.Proto != nil {
			// Extended information about the function
			fmt.Fprint(w, v.Proto)
		} else {
			fmt.Fprint(w, v)
		}
	case *lua.LUserData:
		if jfile, ok := v.Value.(*jpath.JFile); ok {
			fmt.Fprintln(w, v)
			fmt.Fprintf(w, "filename: %s\n", jfile.GetFilename())
			if data, err := jfile.JSON(); err == nil { // success
				fmt.Fprintf(w, "JSON data:\n%s", string(data))
			}
		} else {
			fmt.Fprint(w, v)
		}
	default:
		fmt.Fprint(w, v)
	}
}

// Arguments2buffer retrieves all the arguments given to a Lua function
// and gather the strings in a buffer.
func Arguments2buffer(L *lua.LState, addNewline bool) bytes.Buffer {
	var buf bytes.Buffer
	top := L.GetTop()

	// Add all the string arguments to the buffer
	for i := 1; i <= top; i++ {
		buf.WriteString(L.Get(i).String())
		if i != top {
			buf.WriteString(" ")
		}
	}
	if addNewline {
		buf.WriteString("\n")
	}
	return buf
}

// Strings2table converts a string slice to a Lua table
func Strings2table(L *lua.LState, sl []string) *lua.LTable {
	table := L.NewTable()
	for _, element := range sl {
		table.Append(lua.LString(element))
	}
	return table
}

// Map2table converts a map[string]string to a Lua table
func Map2table(L *lua.LState, m map[string]string) *lua.LTable {
	table := L.NewTable()
	for key, value := range m {
		L.RawSet(table, lua.LString(key), lua.LString(value))
	}
	return table
}

// LValueMaps2table converts a []map[string]lua.LValue to a Lua table
func LValueMaps2table(L *lua.LState, maps []map[string]lua.LValue) *lua.LTable {
	outer := L.NewTable()
	for _, m := range maps {
		inner := L.NewTable()
		for k, v := range m {
			L.RawSet(inner, lua.LString(k), v)
		}
		outer.Append(inner)
	}
	return outer
}

// Table2map converts a Lua table to **one** of the following types, depending
// on the content:
//   map[string]string
//   map[string]int
//   map[int]string
//   map[int]int
// If no suitable keys and values are found, a nil interface is returned.
// If several different types are found, it returns true.
func Table2map(luaTable *lua.LTable, preferInt bool) (interface{}, bool) {

	mapSS, mapSI, mapIS, mapII := Table2maps(luaTable)

	lss := len(mapSS)
	lsi := len(mapSI)
	lis := len(mapIS)
	lii := len(mapII)

	total := lss + lsi + lis + lii

	// Return the first map that has values
	if !preferInt {
		if lss > 0 {
			//log.Println(key, "STRING -> STRING map")
			return interface{}(mapSS), lss < total
		} else if lsi > 0 {
			//log.Println(key, "STRING -> INT map")
			return interface{}(mapSI), lsi < total
		} else if lis > 0 {
			//log.Println(key, "INT -> STRING map")
			return interface{}(mapIS), lis < total
		} else if lii > 0 {
			//log.Println(key, "INT -> INT map")
			return interface{}(mapII), lii < total
		}
	} else {
		if lii > 0 {
			//log.Println(key, "INT -> INT map")
			return interface{}(mapII), lii < total
		} else if lis > 0 {
			//log.Println(key, "INT -> STRING map")
			return interface{}(mapIS), lis < total
		} else if lsi > 0 {
			//log.Println(key, "STRING -> INT map")
			return interface{}(mapSI), lsi < total
		} else if lss > 0 {
			//log.Println(key, "STRING -> STRING map")
			return interface{}(mapSS), lss < total
		}
	}

	return nil, false
}

// Table2maps converts a Lua table to **all** of the following types,
// depending on the content:
//   map[string]string
//   map[string]int
//   map[int]string
//   map[int]int
func Table2maps(luaTable *lua.LTable) (map[string]string, map[string]int, map[int]string, map[int]int) {

	// Initialize possible maps we want to convert to
	mapSS := make(map[string]string)
	mapSI := make(map[string]int)
	mapIS := make(map[int]string)
	mapII := make(map[int]int)

	var skey, svalue lua.LString
	var ikey, ivalue lua.LNumber
	var hasSkey, hasIkey, hasSvalue, hasIvalue bool

	luaTable.ForEach(func(tkey, tvalue lua.LValue) {

		// Convert the keys and values to strings or ints
		skey, hasSkey = tkey.(lua.LString)
		ikey, hasIkey = tkey.(lua.LNumber)
		svalue, hasSvalue = tvalue.(lua.LString)
		ivalue, hasIvalue = tvalue.(lua.LNumber)

		// Store the right keys and values in the right maps
		if hasSkey && hasSvalue {
			mapSS[skey.String()] = svalue.String()
		} else if hasSkey && hasIvalue {
			mapSI[skey.String()] = int(ivalue)
		} else if hasIkey && hasSvalue {
			mapIS[int(ikey)] = svalue.String()
		} else if hasIkey && hasIvalue {
			mapII[int(ikey)] = int(ivalue)
		}
	})

	return mapSS, mapSI, mapIS, mapII
}

// Table2interfaceMap converts a Lua table to a map[string]interface{}
// If values are also tables, they are also attempted converted to map[string]interface{}
func Table2interfaceMap(luaTable *lua.LTable) map[string]interface{} {

	// Even if luaTable.Len() is 0, the table may be full of things

	// Initialize possible maps we want to convert to
	everything := make(map[string]interface{})

	var skey, svalue lua.LString
	var nkey, nvalue lua.LNumber
	var hasSkey, hasSvalue, hasNkey, hasNvalue bool

	luaTable.ForEach(func(tkey, tvalue lua.LValue) {

		// Convert the keys and values to strings or ints or maps
		skey, hasSkey = tkey.(lua.LString)
		nkey, hasNkey = tkey.(lua.LNumber)

		svalue, hasSvalue = tvalue.(lua.LString)
		nvalue, hasNvalue = tvalue.(lua.LNumber)
		secondTableValue, hasTvalue := tvalue.(*lua.LTable)

		// Store the right keys and values in the right maps
		if hasSkey && hasTvalue {
			// Recursive call if the value is another table that can be converted to a string->interface{} map
			everything[skey.String()] = Table2interfaceMap(secondTableValue)
		} else if hasNkey && hasTvalue {
			floatKey := float64(nkey)
			intKey := int(nkey)
			// Use the int key if it's the same as the float representation
			if floatKey == float64(intKey) {
				// Recursive call if the value is another table that can be converted to a string->interface{} map
				everything[fmt.Sprintf("%d", intKey)] = Table2interfaceMap(secondTableValue)
			} else {
				everything[fmt.Sprintf("%f", floatKey)] = Table2interfaceMap(secondTableValue)
			}
		} else if hasSkey && hasSvalue {
			everything[skey.String()] = svalue.String()
		} else if hasSkey && hasNvalue {
			floatVal := float64(nvalue)
			intVal := int(nvalue)
			// Use the int value if it's the same as the float representation
			if floatVal == float64(intVal) {
				everything[skey.String()] = intVal
			} else {
				everything[skey.String()] = floatVal
			}
		} else if hasNkey && hasSvalue {
			floatKey := float64(nkey)
			intKey := int(nkey)
			// Use the int key if it's the same as the float representation
			if floatKey == float64(intKey) {
				everything[fmt.Sprintf("%d", intKey)] = svalue.String()
			} else {
				everything[fmt.Sprintf("%f", floatKey)] = svalue.String()
			}
		} else if hasNkey && hasNvalue {
			var sk, sv string
			floatKey := float64(nkey)
			intKey := int(nkey)
			floatVal := float64(nvalue)
			intVal := int(nvalue)
			// Use the int key if it's the same as the float representation
			if floatKey == float64(intKey) {
				sk = fmt.Sprintf("%d", intKey)
			} else {
				sk = fmt.Sprintf("%f", floatKey)
			}
			// Use the int value if it's the same as the float representation
			if floatVal == float64(intVal) {
				sv = fmt.Sprintf("%d", intVal)
			} else {
				sv = fmt.Sprintf("%f", floatVal)
			}
			everything[sk] = sv
		} else {
			log.Warn("table2interfacemap: Unsupported type for map key. Value:", tvalue)
		}
	})

	return everything
}

// Table2interfaceMapGlua converts a Lua table to a map by using gluamapper.
// If the map really is an array (all the keys are indices), return true.
func Table2interfaceMapGlua(luaTable *lua.LTable) (retmap map[interface{}]interface{}, isArray bool, err error) {
	var (
		m         = make(map[interface{}]interface{})
		opt       = gluamapper.Option{}
		indices   []uint64
		i, length uint64
	)

	// Catch a problem that may occur when converting the map value with gluamapper.ToGoValue
	defer func() {
		if r := recover(); r != nil {
			retmap = m
			err = errToMap // Could not represent Lua structure table as a map
			return
		}
	}()

	// Do the actual conversion
	luaTable.ForEach(func(tkey, tvalue lua.LValue) {
		if i, isNum := tkey.(lua.LNumber); isNum {
			indices = append(indices, uint64(i))
		}
		// If tkey or tvalue is an LTable, give up
		m[gluamapper.ToGoValue(tkey, opt)] = gluamapper.ToGoValue(tvalue, opt)
		length++
	})

	// Report back as a map, not an array, if there are no elements
	if length == 0 {
		return m, false, nil
	}
	// Loop through every index that must be present in an array
	isAnArray := true
	for i = 1; i <= length; i++ {
		// The map must have this index in order to be an array
		hasIt := false
		for _, val := range indices {
			if val == i {
				hasIt = true
				break
			}
		}
		if !hasIt {
			isAnArray = false
			break
		}
	}
	return m, isAnArray, nil
}
