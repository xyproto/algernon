package simplehstore

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
)

// HashMap is a hash map with a name, key and value, stored in PostgreSQL
// Useful when storing several keys and values for a specific username, for instance.
type HashMap dbDatastructure

// NewHashMap creates a new HashMap struct
func NewHashMap(host *Host, name string) (*HashMap, error) {
	h := &HashMap{host, pq.QuoteIdentifier(name)}

	// Create extension hstore
	query := "CREATE EXTENSION hstore"
	// Ignore errors if hstore is already enabled
	h.host.db.Exec(query)

	// Create a new table that maps from the owner string (like user ID) to a blob of hstore ("attr hstore")

	// Using three columns: element id, key and value
	query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s %s, attr hstore)", h.table, ownerCol, defaultStringType)
	if Verbose {
		fmt.Println(query)
	}
	if _, err := h.host.db.Exec(query); err != nil {
		return nil, err
	}
	if Verbose {
		log.Println("Created HSTORE table " + h.table + " in database " + host.dbname)
	}
	return h, nil
}

// CreateIndexTable creates an INDEX table for this hash map, that may speed up lookups
func (h *HashMap) CreateIndexTable() error {
	// strip double quotes from h.table and add _idx at the end
	indexTableName := strings.TrimSuffix(strings.TrimPrefix(h.table, "\""), "\"") + "_idx"
	query := fmt.Sprintf("CREATE INDEX %q ON %s USING GIN (attr)", indexTableName, h.table)
	if Verbose {
		fmt.Println(query)
	}
	_, err := h.host.db.Exec(query)
	return err

}

// RemoveIndexTable removes the INDEX table for this hash map
func (h *HashMap) RemoveIndexTable(owner string) error {
	// strip double quotes from h.table and add _idx at the end
	indexTableName := strings.TrimSuffix(strings.TrimPrefix(h.table, "\""), "\"") + "_idx"
	query := fmt.Sprintf("DROP INDEX %q", indexTableName)
	if Verbose {
		fmt.Println(query)
	}
	_, err := h.host.db.Exec(query)
	return err
}

// Set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (h *HashMap) Set(owner, key, value string) error {
	if !h.host.rawUTF8 {
		Encode(&value)
	}
	encodedValue := value
	// First try updating the key/values
	n, err := h.update(owner, key, encodedValue)
	if err != nil {
		return fmt.Errorf("hashMap Set, update: %s", err)
	}
	// If no rows are affected (SELECTED) by the update, try inserting a row instead
	if n == 0 {
		n, err = h.insert(owner, key, encodedValue)
		if err != nil {
			return fmt.Errorf("hashMap Set, insert: %s", err)
		}
		if n == 0 {
			return errors.New("hashMap Set: could not update or insert any rows")
		}
	}
	// success
	return nil
}

// insert a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (h *HashMap) insert(owner, key, encodedValue string) (int64, error) {
	// Try inserting
	query := fmt.Sprintf("INSERT INTO %s (%s, attr) VALUES ('%s', '\"%s\"=>\"%s\"') ON CONFLICT DO NOTHING", h.table, ownerCol, escapeSingleQuotes(owner), escapeSingleQuotes(key), escapeSingleQuotes(encodedValue))
	if Verbose {
		fmt.Println(query)
	}
	result, err := h.host.db.Exec(query)
	if Verbose {
		log.Println("Inserted row into: "+h.table+" err? ", err)
	}
	n, _ := result.RowsAffected()
	return n, err
}

// update a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (h *HashMap) update(owner, key, encodedValue string) (int64, error) {
	// Try updating
	query := fmt.Sprintf("UPDATE %s SET attr = attr || '%q=>%q' :: hstore WHERE %s = '%s' AND attr ? '%s'", h.table, escapeSingleQuotes(key), escapeSingleQuotes(encodedValue), ownerCol, escapeSingleQuotes(owner), escapeSingleQuotes(key))
	if Verbose {
		fmt.Println(query)
	}
	result, err := h.host.db.Exec(query)
	if Verbose {
		log.Println("Updated row in: "+h.table+" err? ", err)
	}
	if result == nil {
		return 0, fmt.Errorf("no result when trying to update %s -> %s with a value", owner, key)
	}
	n, _ := result.RowsAffected()
	return n, err
}

// SetCheck will set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
// Returns true if the key already existed.
func (h *HashMap) SetCheck(owner, key, value string) (bool, error) {
	if !h.host.rawUTF8 {
		Encode(&value)
	}
	encodedValue := value
	// First try updating the key/values
	n, err := h.update(owner, key, encodedValue)
	if err != nil {
		return false, err
	}
	// If no rows are affected (SELECTED) by the update, try inserting a row instead
	if n == 0 {
		n, err = h.insert(owner, key, encodedValue)
		if err != nil {
			return false, err
		}
		if n == 0 {
			return false, errors.New("could not update or insert any rows")
		}
		return false, nil
	}
	// success, and the key already existed
	return true, nil
}

// Get a value from a hashmap given the element id (for instance a user id) and the key (for instance "password").
func (h *HashMap) Get(owner, key string) (string, error) {
	query := fmt.Sprintf("SELECT attr -> '%s' FROM %s WHERE %s = '%s' AND attr ? '%s'", escapeSingleQuotes(key), h.table, ownerCol, escapeSingleQuotes(owner), escapeSingleQuotes(key))
	if Verbose {
		fmt.Println(query)
	}
	rows, err := h.host.db.Query(query)
	if err != nil {
		return "", err
	}
	if rows == nil {
		return "", errors.New("HashMap Get returned no rows for owner " + owner + " and key " + key)
	}
	defer rows.Close()
	var value sql.NullString
	// Get the value. Should only loop once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			// No rows
			return "", err
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if counter == 0 {
		return "", errors.New("No such owner/key: " + owner + "/" + key)
	}
	s := value.String
	if !h.host.rawUTF8 {
		Decode(&s)
	}
	return s, nil
}

// Has checks if a given owner + key exists in the hash map
func (h *HashMap) Has(owner, key string) (bool, error) {
	query := fmt.Sprintf("SELECT attr -> '%s' FROM %s WHERE %s = '%s' AND attr ? '%s'", escapeSingleQuotes(key), h.table, ownerCol, escapeSingleQuotes(owner), escapeSingleQuotes(key))
	if Verbose {
		fmt.Println(query)
	}
	rows, err := h.host.db.Query(query)
	if err != nil {
		return false, err
	}
	if rows == nil {
		return false, errors.New("HashMap Has returned no rows for owner " + owner)
	}
	defer rows.Close()
	var value sql.NullString
	// Get the value. Should only loop once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			// No rows
			return false, err
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if counter > 1 {
		// Should never happen
		return false, errors.New("Duplicate keys in hash map for owner " + owner + "!")
	}
	return counter > 0, nil
}

// Exists checks if a given owner exists as a hash map at all
func (h *HashMap) Exists(owner string) (bool, error) {
	query := fmt.Sprintf("SELECT attr FROM %s WHERE %s = '%s'", h.table, ownerCol, escapeSingleQuotes(owner))
	rows, err := h.host.db.Query(query)
	if err != nil {
		return false, err
	}
	if rows == nil {
		return false, nil // no rows
	}
	defer rows.Close()
	var value sql.NullString
	counter := 0
	if rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			return false, err // no rows
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return counter > 0, nil // found at least one row
}

// json returns the first found hstore value for the given key as a JSON string
func (h *HashMap) json(owner string) (string, error) {
	query := fmt.Sprintf("SELECT hstore_to_json(hstore(array_agg(altering_pairs))) FROM %s, LATERAL unnest(hstore_to_array(attr)) altering_pairs WHERE %s = '%s'", h.table, ownerCol, escapeSingleQuotes(owner))
	if Verbose {
		fmt.Println(query)
	}
	rows, err := h.host.db.Query(query)
	if err != nil {
		return "", err
	}
	if rows == nil {
		return "", ErrNoAvailableValues
	}
	defer rows.Close()
	var value sql.NullString
	if rows.Next() {
		if err = rows.Scan(&value); err != nil {
			return "", err
		}
		s := value.String
		if !h.host.rawUTF8 {
			Decode(&s)
		}
		// Got a value, return it
		return s, nil
	}
	return "", rows.Err()
}

// All returns all owners for all hash map elements
func (h *HashMap) All() ([]string, error) {
	var (
		values []string
		value  string
	)
	rows, err := h.host.db.Query(fmt.Sprintf("SELECT DISTINCT %s FROM %s", ownerCol, h.table))
	if err != nil {
		return values, err
	}

	if rows == nil {
		return values, ErrNoAvailableValues
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&value)
		if !h.host.rawUTF8 {
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

// AllWhere returns all owner ID's that has a property where key == value
func (h *HashMap) AllWhere(key, value string) ([]string, error) {
	var values []string
	if !h.host.rawUTF8 {
		Encode(&value)
	}
	// Return all owner ID's for all entries that has the given key->value attribute
	//fmt.Printf("SELECT DISTINCT %s FROM %s WHERE attr @> '\"%s\"=>\"%s\"' :: hstore", ownerCol, h.table, key, value)
	rows, err := h.host.db.Query(fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE attr @> '\"%s\"=>\"%s\"' :: hstore", ownerCol, h.table, key, value))
	if err != nil {
		return values, err
	}
	if rows == nil {
		return values, ErrNoAvailableValues
	}
	defer rows.Close()
	var v string
	for rows.Next() {
		err = rows.Scan(&v)
		if !h.host.rawUTF8 {
			Decode(&v)
		}
		values = append(values, v)
		if err != nil {
			return values, err
		}
	}
	err = rows.Err()
	return values, err
}

// Count counts the number of owners for hash map elements
func (h *HashMap) Count() (int, error) {
	var value sql.NullInt32
	rows, err := h.host.db.Query(fmt.Sprintf("SELECT COUNT(*) FROM (SELECT DISTINCT %s FROM %s) as temp", ownerCol, h.table))
	if err != nil {
		return 0, err
	}
	if rows == nil {
		return 0, ErrNoAvailableValues
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&value)
	if err != nil {
		return 0, err
	}
	return int(value.Int32), nil
}

// CountInt64 counts the number of owners for hash map elements
func (h *HashMap) CountInt64() (int64, error) {
	var value sql.NullInt64
	rows, err := h.host.db.Query(fmt.Sprintf("SELECT COUNT(*) FROM (SELECT DISTINCT %s FROM %s) as temp", ownerCol, h.table))
	if err != nil {
		return 0, err
	}
	if rows == nil {
		return 0, ErrNoAvailableValues
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&value)
	if err != nil {
		return 0, err
	}
	return value.Int64, nil
}

// GetAll is deprecated in favor of All
func (h *HashMap) GetAll() ([]string, error) {
	return h.All()
}

// Keys returns all keys for a given owner
func (h *HashMap) Keys(owner string) ([]string, error) {
	rows, err := h.host.db.Query(fmt.Sprintf("SELECT skeys(attr) FROM %s WHERE %s = '%s'", h.table, ownerCol, escapeSingleQuotes(owner)))
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()
	var (
		values []string
		value  string
	)
	for rows.Next() {
		err = rows.Scan(&value)
		values = append(values, value)
		if err != nil {
			// Unusual, worthy of panic
			panic(err.Error())
		}
	}
	if err := rows.Err(); err != nil {
		// Unusual, worthy of panic
		panic(err.Error())
	}
	return values, nil
}

// DelKey removes a key of an owner in a hashmap (for instance the email field for a user)
func (h *HashMap) DelKey(owner, key string) error {
	// Remove a key from the hashmap
	query := fmt.Sprintf("UPDATE %s SET attr = delete(attr, '%s') WHERE attr ? '%s' AND %s = '%s'", h.table, escapeSingleQuotes(key), escapeSingleQuotes(key), ownerCol, escapeSingleQuotes(owner))
	if Verbose {
		fmt.Println(query)
	}
	_, err := h.host.db.Exec(query)
	return err
}

// Del removes an element (for instance a user)
func (h *HashMap) Del(owner string) error {
	// Remove an element id from the table
	results, err := h.host.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = '%s'", h.table, ownerCol, escapeSingleQuotes(owner)))
	if err != nil {
		return err
	}
	n, err := results.RowsAffected()
	if err != nil {
		return err
	}
	if Verbose {
		log.Println(n, "rows were deleted with Del("+owner+")!")
	}
	return nil
}

// Remove this hashmap
func (h *HashMap) Remove() error {
	// Remove the table
	q := fmt.Sprintf("DROP TABLE %s", h.table)
	log.Println(q)
	_, err := h.host.db.Exec(q)
	return err
}

// Clear the contents
func (h *HashMap) Clear() error {
	query := fmt.Sprintf("TRUNCATE TABLE %s", h.table)
	if Verbose {
		fmt.Println(query)
	}
	// Clear the table
	_, err := h.host.db.Exec(query)
	return err
}
