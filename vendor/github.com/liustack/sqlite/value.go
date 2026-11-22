// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

package sqlite

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// timeFormat is how a time.Time is written: SQLite's date and time functions
// read it.
const timeFormat = "2006-01-02 15:04:05.999999999-07:00"

// timeFormats are the forms of a time that a DATE, DATETIME or TIMESTAMP
// column is read in, after timeFormat.
var timeFormats = []string{
	timeFormat,
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
}

func parseTime(s string) (time.Time, bool) {
	s = strings.TrimSuffix(s, "Z")
	for _, f := range timeFormats {
		if t, err := time.ParseInLocation(f, s, time.UTC); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// convert turns an argument into one of the values bind takes: nil, int64,
// float64, bool, []byte, string or time.Time.
func convert(v any) (any, error) {
	switch v := v.(type) {
	case nil, int64, float64, bool, []byte, string, time.Time:
		return v, nil
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint:
		if uint64(v) > 1<<63-1 {
			return nil, fmt.Errorf("sqlite: %d does not fit in an INTEGER", v)
		}
		return int64(v), nil
	case uint64:
		if v > 1<<63-1 {
			return nil, fmt.Errorf("sqlite: %d does not fit in an INTEGER", v)
		}
		return int64(v), nil
	case float32:
		return float64(v), nil
	case driver.Valuer:
		x, err := callValuer(v)
		if err != nil {
			return nil, err
		}
		return convert(x)
	}
	return driver.DefaultParameterConverter.ConvertValue(v)
}

// callValuer calls Value, and gives nil for a nil pointer that implements
// Valuer by its value, as database/sql does.
func callValuer(v driver.Valuer) (driver.Value, error) {
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer && rv.IsNil() && rv.Type().Elem().Implements(reflect.TypeFor[driver.Valuer]()) {
		return nil, nil
	}
	return v.Value()
}
