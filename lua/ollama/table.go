package ollama

import (
	"fmt"

	lua "github.com/xyproto/gopher-lua"
)

// Helper function to convert a Lua table to a slice of float64
func tableToFloatSlice(tbl *lua.LTable) ([]float64, error) {
	var floats []float64
	var errFlag error
	tbl.ForEach(func(_ lua.LValue, data lua.LValue) {
		if errFlag != nil {
			return
		}
		if num, ok := data.(lua.LNumber); ok {
			floats = append(floats, float64(num))
		} else {
			errFlag = fmt.Errorf("table contains non-numeric values")
		}
	})
	if errFlag != nil {
		return []float64{}, errFlag
	}
	return floats, nil
}
