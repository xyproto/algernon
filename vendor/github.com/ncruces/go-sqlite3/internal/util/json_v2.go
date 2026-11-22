//go:build go1.27

package util

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strconv"
	"time"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/errutil"
)

type JSON struct{ Value any }

func (j JSON) Scan(value any) error {
	var buf []byte

	switch v := value.(type) {
	case []byte:
		buf = v
	case string:
		buf = unsafe.Slice(unsafe.StringData(v), len(v))
	case int64:
		buf = strconv.AppendInt(nil, v, 10)
	case float64:
		buf = AppendFloat(nil, v, 64)
	case time.Time:
		buf = append(buf, '"')
		buf = v.AppendFormat(buf, time.RFC3339Nano)
		buf = append(buf, '"')
	case nil:
		buf = []byte("null")
	default:
		panic(errutil.AssertErr())
	}

	return json.Unmarshal(buf, j.Value)
}

func AppendQuote(dst []byte, src string) ([]byte, error) {
	return jsontext.AppendQuote(dst, src)
}
