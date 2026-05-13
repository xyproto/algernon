package convert

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/jpath"
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
			// log.Info("try: for k,v in pairs(t) do pprint(k,v) end")
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
