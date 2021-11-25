package simplehstore

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
)

// List is a list of strings, stored in PostgreSQL
type List dbDatastructure

// NewList creates a new List. Lists are ordered.
func NewList(host *Host, name string) (*List, error) {
	l := &List{host, pq.QuoteIdentifier(name)} // name is the name of the table
	if _, err := l.host.db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id SERIAL PRIMARY KEY, %s %s)", l.table, listCol, defaultStringType)); err != nil {
		if !strings.HasSuffix(err.Error(), "already exists") {
			return nil, err
		}
	}
	if Verbose {
		log.Println("Created table " + l.table + " in database " + host.dbname)
	}
	return l, nil
}

// Add an element to the list
func (l *List) Add(value string) error {
	if !l.host.rawUTF8 {
		Encode(&value)
	}
	_, err := l.host.db.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES ($1)", l.table, listCol), value)
	return err
}

// All retrieves all elements of a list
func (l *List) All() ([]string, error) {
	var (
		values []string
		value  sql.NullString
	)
	rows, err := l.host.db.Query(fmt.Sprintf("SELECT %s FROM %s ORDER BY id", listCol, l.table))
	if err != nil {
		return values, err
	}
	if rows == nil {
		return values, ErrNoAvailableValues
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&value)
		s := value.String
		if !l.host.rawUTF8 {
			Decode(&s)
		}
		values = append(values, s)
		if err != nil {
			return values, err
		}
	}
	err = rows.Err()
	return values, err
}

// Has checks if an element exists in the list
func (l *List) Has(owner string) (bool, error) {
	rows, err := l.host.db.Query(fmt.Sprintf("SELECT %s FROM %s WHERE id = '%s'", listCol, l.table, owner))
	if err != nil {
		return false, err
	}
	if rows == nil {
		return false, ErrNoAvailableValues
	}
	return true, nil
}

// GetAll is deprecated in favor of All
func (l *List) GetAll() ([]string, error) {
	return l.All()
}

// Last retrieves the last element of a list
func (l *List) Last() (string, error) {
	var value sql.NullString
	// Fetches the item with the largest id.
	// Faster than "ORDER BY id DESC limit 1" for large tables.
	rows, err := l.host.db.Query(fmt.Sprintf("SELECT %s FROM %s WHERE id = (SELECT MAX(id) FROM %s)", listCol, l.table, l.table))
	if err != nil {
		return "", err
	}
	if rows == nil {
		return "", ErrNoAvailableValues
	}
	defer rows.Close()
	// Get the value. Will only loop once.
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	s := value.String
	if !l.host.rawUTF8 {
		Decode(&s)
	}
	return s, nil
}

// GetLast is deprecated in favor of Last
func (l *List) GetLast() (string, error) {
	return l.Last()
}

// LastN retrieves the N last elements of a list. If there are too few
// available elements, the values that were found are returned, together
// with a TooFewElementsError.
func (l *List) LastN(n int) ([]string, error) {
	var (
		values []string
		value  string
	)
	rows, err := l.host.db.Query(fmt.Sprintf("SELECT %s FROM (SELECT * FROM %s ORDER BY id DESC limit %d)sub ORDER BY id ASC", listCol, l.table, n))
	if err != nil {
		return values, err
	}
	if rows == nil {
		return values, ErrNoAvailableValues
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&value)
		if !l.host.rawUTF8 {
			Decode(&value)
		}
		values = append(values, value)
		if err != nil {
			return values, err
		}
	}
	if err := rows.Err(); err != nil {
		return values, err
	}
	if len(values) < n {
		return values, ErrTooFewResults
	}
	return values, nil
}

// GetLastN is deprecated in favor of LastN
func (l *List) GetLastN(n int) ([]string, error) {
	return l.LastN(n)
}

// RemoveByIndex can remove the Nth item, in the same order as returned by All()
func (l *List) RemoveByIndex(index int) error {
	_, err := l.host.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id IN (SELECT id FROM %s ORDER BY id LIMIT 1 OFFSET %d)", l.table, l.table, index))
	return err
}

// Remove this list
func (l *List) Remove() error {
	// Remove the table
	_, err := l.host.db.Exec(fmt.Sprintf("DROP TABLE %s", l.table))
	return err
}

// Clear the list contents
func (l *List) Clear() error {
	// Clear the table
	_, err := l.host.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", l.table))
	return err
}

// Count counts the number of elements in this list
func (l *List) Count() (int, error) {
	var value sql.NullInt32
	rows, err := l.host.db.Query(fmt.Sprintf("SELECT COUNT(*) FROM (SELECT DISTINCT %s FROM %s) as temp", listCol, l.table))
	if err != nil {
		return 0, err
	}
	if rows == nil {
		return 0, ErrNoAvailableValues
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			return 0, err
		}
	}
	return int(value.Int32), nil
}

// CountInt64 counts the number of elements in this list (int64)
func (l *List) CountInt64() (int64, error) {
	var value sql.NullInt64
	rows, err := l.host.db.Query(fmt.Sprintf("SELECT COUNT(*) FROM (SELECT DISTINCT %s FROM %s) as temp", listCol, l.table))
	if err != nil {
		return 0, err
	}
	if rows == nil {
		return 0, ErrNoAvailableValues
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			return 0, err
		}
	}
	return value.Int64, nil
}
