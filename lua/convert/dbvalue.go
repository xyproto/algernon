package convert

import (
	"encoding/json"
	"fmt"
	"time"

	lua "github.com/xyproto/gopher-lua"
)

// LValueWrapper decorates lua.LValue to help retrieve values from the database.
type LValueWrapper struct {
	LValue lua.LValue
}

// Scan implements the sql.Scanner interface for database deserialization.
func (w *LValueWrapper) Scan(value any) error {
	if value == nil {
		*w = LValueWrapper{lua.LNil}
		return nil
	}

	switch v := value.(type) {
	case float32:
		*w = LValueWrapper{lua.LNumber(float64(v))}
	case float64:
		*w = LValueWrapper{lua.LNumber(v)}
	case int64:
		*w = LValueWrapper{lua.LNumber(float64(v))}
	case string:
		*w = LValueWrapper{lua.LString(v)}
	case []byte:
		*w = LValueWrapper{lua.LString(string(v))}
	case time.Time:
		*w = LValueWrapper{lua.LNumber(float64(v.Unix()))}
	default:
		return fmt.Errorf("unable to scan type %T into lua value wrapper", value)
	}

	return nil
}

// LValueWrappers is a convenience type to easily map to a slice of lua.LValue
type LValueWrappers []LValueWrapper

// Unwrap produces a slice of lua.LValue from the given LValueWrappers
func (w LValueWrappers) Unwrap() (s []lua.LValue) {
	s = make([]lua.LValue, len(w))
	for i, v := range w {
		s[i] = v.LValue
	}
	return
}

// Interfaces returns a slice of any values from the given LValueWrappers
func (w LValueWrappers) Interfaces() (s []any) {
	s = make([]any, len(w))
	for i := range w {
		s[i] = &w[i]
	}
	return
}

// ExtractParams converts a Lua table at the given stack index to a slice of query parameters
func ExtractParams(L *lua.LState, idx int) []any {
	var params []any
	table := L.ToTable(idx)
	if table == nil {
		return params
	}
	table.ForEach(func(k lua.LValue, v lua.LValue) {
		switch val := v.(type) {
		case lua.LNumber:
			params = append(params, float64(val))
		case lua.LString:
			params = append(params, string(val))
		case *lua.LNilType:
			params = append(params, nil)
		case lua.LBool:
			params = append(params, bool(val))
		default:
			params = append(params, v.String())
		}
	})
	return params
}

// Table2JSON converts a Lua table to a JSON string
func Table2JSON(L *lua.LState, table *lua.LTable) (string, error) {
	m := make(map[string]any)
	table.ForEach(func(k lua.LValue, v lua.LValue) {
		key := k.String()
		switch val := v.(type) {
		case lua.LNumber:
			m[key] = float64(val)
		case lua.LString:
			m[key] = string(val)
		case *lua.LNilType:
			m[key] = nil
		case lua.LBool:
			m[key] = bool(val)
		default:
			m[key] = v.String()
		}
	})
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// JSON2table converts a JSON string to a Lua table
func JSON2table(L *lua.LState, data string) *lua.LTable {
	table := L.NewTable()
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return table
	}
	for k, v := range m {
		switch val := v.(type) {
		case float64:
			L.RawSet(table, lua.LString(k), lua.LNumber(val))
		case string:
			L.RawSet(table, lua.LString(k), lua.LString(val))
		case bool:
			L.RawSet(table, lua.LString(k), lua.LBool(val))
		case nil:
			L.RawSet(table, lua.LString(k), lua.LNil)
		default:
			L.RawSet(table, lua.LString(k), lua.LString(fmt.Sprintf("%v", val)))
		}
	}
	return table
}
