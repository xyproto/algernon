// Package simplehstore offers a simple way to use a PostgreSQL database with HSTORE.
// The database backend is interchangeable with Redis (xyproto/simpleredis), BoltDB (xyproto/simplebolt) and
// Mariadb/MySQL (xyproto/simplemaria) since the xyproto/pinterface packages is used.
package simplehstore

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	// Using the PostgreSQL database engine
	pq "github.com/lib/pq"
)

const (
	// Version number. Stable API within major version numbers.
	Version = 2.1
)

// Host represents a PostgreSQL database
type Host struct {
	db     *sql.DB
	dbname string

	// If set to true, any UTF-8 string will be let through as it is.
	// Some UTF-8 strings may be unpalatable for PostgreSQL when performing
	// SQL queries. The default is "false".
	rawUTF8 bool
}

// Common for each of the db data structures used here
type dbDatastructure struct {
	host  *Host
	table string
}

type (
	List     dbDatastructure
	Set      dbDatastructure
	HashMap  dbDatastructure
	KeyValue dbDatastructure
)

const (

	// The default "username:password@host:port/database" that the database is running at
	defaultDatabaseServer = "postgres:@127.0.0.1/" // "username:password@server:port/"
	defaultDatabaseName   = "test"                 // "main"
	defaultStringType     = "TEXT"
	defaultPort           = 5432

	encoding = "UTF8"

	// Column names
	listCol  = "a_list"
	setCol   = "a_set"
	ownerCol = "owner"
	kvPrefix = "a_kv_"
)

// Test if the local database server is up and running.
func TestConnection() (err error) {
	return TestConnectionHost(defaultDatabaseServer)
}

// Test if a given database server is up and running.
// connectionString may be on the form "username:password@host:port/database".
func TestConnectionHost(connectionString string) (err error) {
	newConnectionString, _ := rebuildConnectionString(connectionString)
	// Connect to the given host:port
	db, err := sql.Open("postgres", newConnectionString)
	defer db.Close()
	err = db.Ping()
	if Verbose {
		if err != nil {
			log.Println("Ping: failed")
		} else {
			log.Println("Ping: ok")
		}
	}
	return err
}

// TestConnectionHostWithDSN checks if a given database server is up and running.
func TestConnectionHostWithDSN(connectionString string) (err error) {
	// Connect to the given host:port
	db, err := sql.Open("postgres", connectionString)
	defer db.Close()
	err = db.Ping()
	if Verbose {
		if err != nil {
			log.Println("Ping: failed")
		} else {
			log.Println("Ping: ok")
		}
	}
	return err
}

// Enclose in single quote and escape single quotes within
func singleQuote(s string) string {
	return "'" + escape(s) + "'"
}

// Escape single quotes
func escape(s string) string {
	return strings.Replace(s, "'", "''", -1)
}

/* --- Host functions --- */

// NewHost sets up a new database connection.
// connectionString may be on the form "username:password@host:port/database".
func NewHost(connectionString string) *Host {
	newConnectionString, dbname := rebuildConnectionString(connectionString)
	db, err := sql.Open("postgres", newConnectionString)
	if err != nil {
		log.Fatalln("Could not connect to " + newConnectionString + "!")
	}
	host := &Host{db, pq.QuoteIdentifier(dbname), false}
	if err := host.Ping(); err != nil {
		log.Fatalln("Host does not reply to ping: " + err.Error())
	}
	if err := host.createDatabase(); err != nil {
		log.Fatalln("Could not create database " + host.dbname + ": " + err.Error())
	}
	if err := host.useDatabase(); err != nil {
		panic("Could not use database " + host.dbname + ": " + err.Error())
	}
	return host
}

// NewHostWithDSN creates a new database connection with a valid DSN.
func NewHostWithDSN(connectionString string, dbname string) *Host {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalln("Could not connect to " + connectionString + "!")
	}
	host := &Host{db, pq.QuoteIdentifier(dbname), false}
	if err := host.Ping(); err != nil {
		log.Fatalln("Host does not reply to ping: " + err.Error())
	}
	if err := host.createDatabase(); err != nil {
		log.Fatalln("Could not create database " + host.dbname + ": " + err.Error())
	}
	if err := host.useDatabase(); err != nil {
		panic("Could not use database " + host.dbname + ": " + err.Error())
	}
	return host
}

// New sets up a connection to the default (local) database host
func New() *Host {
	connectionString := defaultDatabaseServer + defaultDatabaseName
	if !strings.HasSuffix(defaultDatabaseServer, "/") {
		connectionString = defaultDatabaseServer + "/" + defaultDatabaseName
	}
	return NewHost(connectionString)
}

// SetRawUTF8 can be used to select if the UTF-8 data be unprocessed, and not
// hex encoded and compressed. Unprocessed UTF-8 may be slightly faster,
// but malformed UTF-8 strings can potentially cause problems.
// Encoding the strings before sending them to PostgreSQL is the default.
// Choose the setting that best suits your situation.
func (host *Host) SetRawUTF8(enabled bool) {
	host.rawUTF8 = enabled
}

// SelectDatabase sets a different database name and creates the database if needed.
func (host *Host) SelectDatabase(dbname string) error {
	host.dbname = dbname
	if err := host.createDatabase(); err != nil {
		return err
	}
	if err := host.useDatabase(); err != nil {
		return err
	}
	return nil
}

// Will create the database if it does not already exist
func (host *Host) createDatabase() error {
	if _, err := host.db.Exec(fmt.Sprintf("CREATE DATABASE %s WITH ENCODING '%s'", host.dbname, encoding)); err != nil {
		if !strings.HasSuffix(err.Error(), "already exists") {
			return err
		}
	}
	// Ignore the error if HSTORE has already been enabled
	if _, err := host.db.Exec("CREATE EXTENSION hstore"); err == nil {
		if Verbose {
			log.Println("Enabled HSTORE")
		}
	}
	return nil
}

// Use the host.dbname database
func (host *Host) useDatabase() error {
	if Verbose {
		log.Println("Using database " + host.dbname)
	}
	return nil
}

func (host *Host) Database() *sql.DB {
	return host.db
}

// Close the connection
func (host *Host) Close() {
	host.db.Close()
}

// Ping the host
func (host *Host) Ping() error {
	return host.db.Ping()
}

/* --- List functions --- */

// Create a new list. Lists are ordered.
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

// Get all elements of a list
func (l *List) GetAll() ([]string, error) {
	var (
		values []string
		value  string
	)
	rows, err := l.host.db.Query(fmt.Sprintf("SELECT %s FROM %s ORDER BY id", listCol, l.table))
	if err != nil {
		return values, err
	}
	if rows == nil {
		return values, errors.New("List GetAll returned no rows")
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
	return values, nil
}

// Get the last element of a list
func (l *List) GetLast() (string, error) {
	var value string
	// Fetches the item with the largest id.
	// Faster than "ORDER BY id DESC limit 1" for large tables.
	rows, err := l.host.db.Query(fmt.Sprintf("SELECT %s FROM %s WHERE id = (SELECT MAX(id) FROM %s)", listCol, l.table, l.table))
	if err != nil {
		return value, err
	}
	if rows == nil {
		return value, errors.New("List GetLast returned no rows")
	}
	defer rows.Close()
	// Get the value. Will only loop once.
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			return value, err
		}
	}
	if err := rows.Err(); err != nil {
		return value, err
	}
	if !l.host.rawUTF8 {
		Decode(&value)
	}
	return value, nil
}

// Get the last N elements of a list
func (l *List) GetLastN(n int) ([]string, error) {
	var (
		values []string
		value  string
	)
	rows, err := l.host.db.Query(fmt.Sprintf("SELECT %s FROM (SELECT * FROM %s ORDER BY id DESC limit %d)sub ORDER BY id ASC", listCol, l.table, n))
	if err != nil {
		return values, err
	}
	if rows == nil {
		return values, errors.New("List GetLastN returned no rows for n " + strconv.Itoa(n))
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
		return values, errors.New("Too few elements in table at GetLastN")
	}
	return values, nil
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

/* --- Set functions --- */

// Create a new set
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

// Check if a given value is in the set
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
	var scanValue string
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
	if counter > 1 {
		// Should never happen
		return false, errors.New("Duplicate keys in set for value " + value + "!")
	}
	return counter > 0, nil
}

// Get all elements of the set
func (s *Set) GetAll() ([]string, error) {
	var (
		values []string
		value  string
	)
	rows, err := s.host.db.Query(fmt.Sprintf("SELECT %s FROM %s", setCol, s.table))
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
	if err := rows.Err(); err != nil {
		return values, err
	}
	return values, nil
}

// Remove an element from the set
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

/* --- HashMap functions --- */

// Create a new hashmap
func NewHashMap(host *Host, name string) (*HashMap, error) {
	h := &HashMap{host, pq.QuoteIdentifier(name)}
	// Using three columns: element id, key and value
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s %s, attr hstore)", h.table, ownerCol, defaultStringType)
	if _, err := h.host.db.Exec(query); err != nil {
		if !strings.HasSuffix(err.Error(), "already exists") {
			return nil, err
		}
	}
	if Verbose {
		log.Println("Created HSTORE table " + h.table + " in database " + host.dbname)
	}
	return h, nil
}

// Set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (h *HashMap) Set(owner, key, value string) error {
	// See if the owner and key already exists
	hasKey, err := h.Has(owner, key)
	if err != nil {
		return err
	}
	if Verbose {
		log.Printf("%s/%s exists? %v\n", owner, key, hasKey)
	}
	if !h.host.rawUTF8 {
		Encode(&value)
	}
	if hasKey {
		_, err = h.host.db.Exec(fmt.Sprintf("UPDATE %s SET attr = attr || '\"%s\"=>\"%s\"' :: hstore WHERE %s = %s AND attr ? %s", h.table, escape(key), escape(value), ownerCol, singleQuote(owner), singleQuote(key)))
		if Verbose {
			log.Println("Updated HSTORE table: " + h.table)
		}
	} else {
		_, err = h.host.db.Exec(fmt.Sprintf("INSERT INTO %s (%s, attr) VALUES (%s, '\"%s\"=>\"%s\"')", h.table, ownerCol, singleQuote(owner), escape(key), escape(value)))
		if Verbose {
			log.Println("Added to HSTORE table: " + h.table)
		}
	}
	return err
}

// Get a value from a hashmap given the element id (for instance a user id) and the key (for instance "password").
func (h *HashMap) Get(owner, key string) (string, error) {
	rows, err := h.host.db.Query(fmt.Sprintf("SELECT attr -> %s FROM %s WHERE %s = %s AND attr ? %s", singleQuote(key), h.table, ownerCol, singleQuote(owner), singleQuote(key)))
	if err != nil {
		return "", err
	}
	if rows == nil {
		return "", errors.New("HashMap Get returned no rows for owner " + owner + " and key " + key)
	}
	defer rows.Close()
	var value string
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
	if !h.host.rawUTF8 {
		Decode(&value)
	}
	return value, nil
}

// Check if a given owner + key is in the hash map
func (h *HashMap) Has(owner, key string) (bool, error) {
	rows, err := h.host.db.Query(fmt.Sprintf("SELECT attr -> %s FROM %s WHERE %s = %s AND attr ? %s", singleQuote(key), h.table, ownerCol, singleQuote(owner), singleQuote(key)))
	if err != nil {
		return false, err
	}
	if rows == nil {
		return false, errors.New("HashMap Has returned no rows for owner " + owner)
	}
	defer rows.Close()
	var value string
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

// Check if a given owner exists as a hash map at all
func (h *HashMap) Exists(owner string) (bool, error) {
	rows, err := h.host.db.Query(fmt.Sprintf("SELECT attr FROM %s WHERE %s = %s", h.table, ownerCol, singleQuote(owner)))
	if err != nil {
		return false, err
	}
	if rows == nil {
		return false, errors.New("HashMap Exists returned no rows for owner " + owner)
	}
	defer rows.Close()
	var value string
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
	return counter > 0, nil
}

// Get all owners for all hash elements
func (h *HashMap) GetAll() ([]string, error) {
	var (
		values []string
		value  string
	)
	rows, err := h.host.db.Query(fmt.Sprintf("SELECT %s FROM %s", ownerCol, h.table))
	if err != nil {
		return values, err
	}
	if rows == nil {
		return values, errors.New("HashMap GetAll returned no rows")
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
	if err := rows.Err(); err != nil {
		return values, err
	}
	return values, nil
}

// Remove a key for an entry in a hashmap (for instance the email field for a user)
func (h *HashMap) DelKey(owner, key string) error {
	// Remove a key from the hashmap
	_, err := h.host.db.Exec(fmt.Sprintf("UPDATE %s SET attr = delete(attr, %s) WHERE %s = %s AND attr ? %s", h.table, singleQuote(key), ownerCol, singleQuote(owner), singleQuote(key)))
	return err
}

// Remove an element (for instance a user)
func (h *HashMap) Del(owner string) error {
	// Remove an element id from the table
	results, err := h.host.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = %s", h.table, ownerCol, singleQuote(owner)))
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
	_, err := h.host.db.Exec(fmt.Sprintf("DROP TABLE %s", h.table))
	return err
}

// Clear the contents
func (h *HashMap) Clear() error {
	// Clear the table
	_, err := h.host.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", h.table))
	return err
}

/* --- KeyValue functions --- */

// Create a new key/value
func NewKeyValue(host *Host, name string) (*KeyValue, error) {
	kv := &KeyValue{host, name}
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (attr hstore)", pq.QuoteIdentifier(kvPrefix+kv.table))
	if _, err := kv.host.db.Exec(query); err != nil {
		if !strings.HasSuffix(err.Error(), "already exists") {
			return nil, err
		}
	}
	if Verbose {
		log.Println("Created HSTORE table " + pq.QuoteIdentifier(kvPrefix+kv.table) + " in database " + host.dbname)
	}
	return kv, nil
}

// Set a key and value
func (kv *KeyValue) Set(key, value string) error {
	if !kv.host.rawUTF8 {
		Encode(&value)
	}
	if _, err := kv.Get(key); err != nil {
		// Key does not exist, create it
		_, err = kv.host.db.Exec(fmt.Sprintf("INSERT INTO %s (attr) VALUES ('\"%s\"=>\"%s\"')", pq.QuoteIdentifier(kvPrefix+kv.table), escape(key), escape(value)))
		return err
	}
	// Key exists, update the value
	_, err := kv.host.db.Exec(fmt.Sprintf("UPDATE %s SET attr = attr || '\"%s\"=>\"%s\"' :: hstore", pq.QuoteIdentifier(kvPrefix+kv.table), escape(key), escape(value)))
	return err
}

// Get a value given a key
func (kv *KeyValue) Get(key string) (string, error) {
	rows, err := kv.host.db.Query(fmt.Sprintf("SELECT attr -> %s FROM %s", singleQuote(key), pq.QuoteIdentifier(kvPrefix+kv.table)))
	if err != nil {
		return "", err
	}
	if rows == nil {
		return "", errors.New("KeyValue Get returned no rows for key " + key)
	}
	defer rows.Close()
	var value string
	// Get the value. Should only loop once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			return "", err
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if counter != 1 {
		return "", errors.New("Wrong number of keys in KeyValue table: " + kvPrefix + kv.table)
	}
	if !kv.host.rawUTF8 {
		Decode(&value)
	}
	return value, nil
}

// Inc increases the value of a key and returns the new value.
// Returns "1" if no previous value is found.
func (kv *KeyValue) Inc(key string) (string, error) {
	// Retrieve the current value, if any
	num := 0
	// See if we can fetch an existing value. NOTE: "== nil"
	if val, err := kv.Get(key); err == nil {
		// See if we can convert the value to a number. NOTE: "== nil"
		if converted, errConv := strconv.Atoi(val); errConv == nil {
			num = converted
		}
	} else {
		// The key does not exist, create a new one.
		// This is to reflect the behavior of INCR in Redis.
		NewKeyValue(kv.host, kv.table)
	}
	// Num is now either 0 or the previous numeric value
	num++
	// Convert the new value to a string
	val := strconv.Itoa(num)
	// Store the new number
	if err := kv.Set(key, val); err != nil {
		// Saving the value failed
		return "0", err
	}
	// Success
	return val, nil
}

// Remove a key
func (kv *KeyValue) Del(key string) error {
	_, err := kv.host.db.Exec(fmt.Sprintf("UPDATE %s SET attr = delete(attr, %s)", pq.QuoteIdentifier(kvPrefix+kv.table), singleQuote(key)))
	return err
}

// Remove this key/value
func (kv *KeyValue) Remove() error {
	// Remove the table
	_, err := kv.host.db.Exec(fmt.Sprintf("DROP TABLE %s", pq.QuoteIdentifier(kvPrefix+kv.table)))
	return err
}

// Clear this key/value
func (kv *KeyValue) Clear() error {
	// Remove the table
	_, err := kv.host.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", pq.QuoteIdentifier(kvPrefix+kv.table)))
	return err
}
