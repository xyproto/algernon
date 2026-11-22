// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build go1.27 && (((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64)))

// Go 1.27 lets a driver store a column in the destination of Rows.Scan
// itself (driver.RowsColumnScanner). With an older Go, database/sql takes
// the values from Next and converts them: the same results, with an
// interface value for each column.

package sqlite

import (
	"database/sql"
	"database/sql/driver"

	"github.com/liustack/sqlite/internal/csqlite"
)

var _ driver.RowsColumnScanner = (*rows)(nil)

// ScanColumn stores column i of the row in dest. The common pairs of
// column type and destination are stored directly, without putting the
// value in an interface. Everything else, and every column of a time type,
// goes through sql.ConvertAssign with the value Next gives, so both ways
// give the same results.
func (r *rows) ScanColumn(ctx driver.ScanContext, i int, dest any) error {
	s := r.s.s
	if !r.isTime(i) {
		switch s.ColumnType(i) {
		case csqlite.TypeInteger:
			switch d := dest.(type) {
			case *int64:
				*d = s.ColumnInt64(i)
				return nil
			case *int:
				*d = int(s.ColumnInt64(i)) // int is 64 bits on every target
				return nil
			case *sql.NullInt64:
				d.Int64, d.Valid = s.ColumnInt64(i), true
				return nil
			}
		case csqlite.TypeFloat:
			switch d := dest.(type) {
			case *float64:
				*d = s.ColumnDouble(i)
				return nil
			case *sql.NullFloat64:
				d.Float64, d.Valid = s.ColumnDouble(i), true
				return nil
			}
		case csqlite.TypeText:
			switch d := dest.(type) {
			case *string:
				v, err := s.ColumnText(i)
				*d = v
				return err
			case *sql.NullString:
				v, err := s.ColumnText(i)
				d.String, d.Valid = v, err == nil
				return err
			}
		}
	}
	v, err := r.value(i)
	if err != nil {
		return err
	}
	return sql.ConvertAssign(ctx, dest, v)
}
