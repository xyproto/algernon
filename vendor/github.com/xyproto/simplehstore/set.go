package simplehstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
)

// Set is a set of strings, stored in PostgreSQL
type Set dbDatastructure

// NewSet creates a new set
func NewSet(host *Host, name string) (*Set, error) {
	s := &Set{host, pq.QuoteIdentifier(name)} // name is the name of the table
	// list is the name of the column
	if _, err := s.host.db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s %s)", s.table, setCol, defaultStringType)); err != nil {
		if !strings.HasSuffix(err.Error(), "already exists") {
			return nil, err
		}
	}
	if Verbose {
		log.Println("Created table " + s.table + " in database " + host.dbname)
	}
	return s, nil
}

// Add an element to the set
func (s *Set) Add(value string) error {
	originalValue := value
	if !s.host.rawUTF8 {
		Encode(&value)
	}
	// Check that the value is not already there before adding
	has, err := s.Has(originalValue)
	if !has && (err == nil) {
		_, err = s.host.db.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES ($1)", s.table, setCol), value)
	}
	return err
}

// add an element to the set, as part of a transaction
func (s *Set) addWithTransaction(ctx context.Context, transaction *sql.Tx, value string) error {
	originalValue := value
	if !s.host.rawUTF8 {
		Encode(&value)
	}
	// Check that the value is not already there before adding
	has, err := s.Has(originalValue)
	if !has && (err == nil) {
		_, err = transaction.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES ($1)", s.table, setCol), value)
	}
	return err
}

// Has checks if the given value is in the set
func (s *Set) Has(value string) (bool, error) {
	if !s.host.rawUTF8 {
		Encode(&value)
	}
	rows, err := s.host.db.Query(fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", setCol, s.table, setCol), value)
	if err != nil {
		return false, err
	}
	if rows == nil {
		return false, errors.New("Set Has returned no rows for value " + value)
	}
	defer rows.Close()
	var scanValue sql.NullString
	// Get the value. Should not loop more than once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&scanValue)
		if err != nil {
			// No rows
			return false, err
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	//if counter > 1 {
	// more than one element that has the same *value* is fine!
	//}
	return counter > 0, nil
}

// All returns all elements in the set
func (s *Set) All() ([]string, error) {
	var (
		values []string
		value  string
	)
	rows, err := s.host.db.Query(fmt.Sprintf("SELECT DISTINCT %s FROM %s", setCol, s.table))
	if err != nil {
		return values, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&value)
		if !s.host.rawUTF8 {
			Decode(&value)
		}
		values = append(values, value)
		if err != nil {
			return values, err
		}
	}
	err = rows.Err()
	return values, err
}

// GetAll is deprecated in favor of All
func (s *Set) GetAll() ([]string, error) {
	return s.All()
}

// Del removes an element from the set
func (s *Set) Del(value string) error {
	if !s.host.rawUTF8 {
		Encode(&value)
	}
	// Remove a value from the table
	_, err := s.host.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = '%s'", s.table, setCol, value))
	return err
}

// Remove this set
func (s *Set) Remove() error {
	// Remove the table
	_, err := s.host.db.Exec(fmt.Sprintf("DROP TABLE %s", s.table))
	return err
}

// Clear the list contents
func (s *Set) Clear() error {
	// Clear the table
	_, err := s.host.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", s.table))
	return err
}

// Count counts the number of elements in this list
func (s *Set) Count() (int, error) {
	var value sql.NullInt32
	rows, err := s.host.db.Query(fmt.Sprintf("SELECT COUNT(*) FROM (SELECT DISTINCT %s FROM %s) as temp", setCol, s.table))
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
func (s *Set) CountInt64() (int64, error) {
	var value sql.NullInt64
	rows, err := s.host.db.Query(fmt.Sprintf("SELECT COUNT(*) FROM (SELECT DISTINCT %s FROM %s) as temp", setCol, s.table))
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
