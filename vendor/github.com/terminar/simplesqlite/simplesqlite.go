// Package simplesqlite offers a simple way to use a SQlite database.
// This database backend is interchangeable with xyproto/simpleredis, xyproto/simplemaria
// xyproto/simplebolt, since they all use xyproto/pinterface.
// Based on xyproto/simplemaria

package simplesqlite

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/xyproto/env/v2"

	"database/sql"

	_ "github.com/mattn/go-sqlite3" //sqlite3
)

const (
	// Version number. Stable API within major version numbers.
	Version = 3.2
)

// File represents a specific database
type File struct {
	db *sql.DB
}

// Common for each of the db datastructures used here
type dbDatastructure struct {
	file  *File
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
	defaultDatabaseFile = "sqlite.db"
	defaultStringType   = "TEXT" // "VARCHAR(65535)"

	// Column names
	listCol  = "a_list"
	setCol   = "a_set"
	keyCol   = "property"
	valCol   = "value"
	ownerCol = "owner"
)

// Test if the local database server is up and running.
func TestConnection() (err error) {
	return TestConnectionFile(defaultDatabaseFile)
}

// Test if a given database server is up and running.
// connectionString may be on the form "sqlite.db&cache=shared&mode=memory".
func TestConnectionFile(connectionString string) (err error) {
	// Connect to the given string
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return err
	}
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

/* --- File functions --- */

// Create a new database connection.
// connectionString may be on the form "sqlite.db&cache=shared&mode=memory".
func NewFile(connectionString string) *File {
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		log.Fatalln("Could not open " + connectionString + "!")
	}
	file := &File{db}
	if err := file.Ping(); err != nil {
		log.Fatalln("File does not reply to ping: " + err.Error())
	}

	return file
}

// The default database connection
func New() *File {
	filename := env.Str("SQLITE_FILE", defaultDatabaseFile)

	return NewFile(filename)
}

// Close the connection
func (file *File) Close() {
	file.db.Close()
}

// Ping the file
func (file *File) Ping() error {
	return file.db.Ping()
}

/* --- List functions --- */

// Create a new list. Lists are ordered.
func NewList(file *File, name string) (*List, error) {
	l := &List{file, name}
	if _, err := l.file.db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (id INTEGER PRIMARY KEY, " + listCol + " " + defaultStringType + ")"); err != nil {
		return nil, err
	}
	if Verbose {
		log.Println("Created table " + name + " in database")
	}
	return l, nil
}

// Add an element to the list
func (l *List) Add(value string) error {

	_, err := l.file.db.Exec("INSERT INTO "+l.table+" ("+listCol+") VALUES (?)", value)
	return err
}

// Get all elements of a list
func (l *List) All() ([]string, error) {
	rows, err := l.file.db.Query("SELECT " + listCol + " FROM " + l.table + " ORDER BY id")
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

// Deprecated, please use .All() instead
func (l *List) GetAll() ([]string, error) {
	return l.All()
}

// Get the last element of a list
func (l *List) Last() (string, error) {
	// Fetches the item with the largest id.
	// Faster than "ORDER BY id DESC limit 1" for large tables.
	rows, err := l.file.db.Query("SELECT " + listCol + " FROM " + l.table + " WHERE id = (SELECT MAX(id) FROM " + l.table + ")")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var value string
	// Get the value. Will only loop once.
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			// Unusual, worthy of panic
			panic(err.Error())
		}
	}
	if err := rows.Err(); err != nil {
		// Unusual, worthy of panic
		panic(err.Error())
	}

	return value, nil
}

// Deprecated, please use .Last() instead
func (l *List) GetLast() (string, error) {
	return l.Last()
}

// Get the last N elements of a list
func (l *List) LastN(n int) ([]string, error) {
	rows, err := l.file.db.Query("SELECT " + listCol + " FROM (SELECT * FROM " + l.table + " ORDER BY id DESC limit " + strconv.Itoa(n) + ")sub ORDER BY id ASC")
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
	if len(values) < n {
		return []string{}, errors.New("Too few elements in table at GetLastN")
	}
	return values, nil
}

// Deprecated, please use .LastN(n) instead
func (l *List) GetLastN(n int) ([]string, error) {
	return l.LastN(n)
}

// Remove this list
func (l *List) Remove() error {
	// Remove the table
	_, err := l.file.db.Exec("DROP TABLE " + l.table)
	return err
}

// Clear the list contents
func (l *List) Clear() error {
	// Clear the table
	_, err := l.file.db.Exec("TRUNCATE TABLE " + l.table)
	return err
}

/* --- Set functions --- */

// Create a new set
func NewSet(file *File, name string) (*Set, error) {
	s := &Set{file, name}
	if _, err := s.file.db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (" + setCol + " " + defaultStringType + ")"); err != nil {
		return nil, err
	}
	if Verbose {
		log.Println("Created table " + name + " in database")
	}
	return s, nil
}

// Add an element to the set
func (s *Set) Add(value string) error {
	originalValue := value

	// Check if the value is not already there before adding
	has, err := s.Has(originalValue)
	if !has && (err == nil) {
		_, err = s.file.db.Exec("INSERT INTO "+s.table+" ("+setCol+") VALUES (?)", value)
	}
	return err
}

// Check if a given value is in the set
func (s *Set) Has(value string) (bool, error) {

	rows, err := s.file.db.Query("SELECT "+setCol+" FROM "+s.table+" WHERE "+setCol+" = ?", value)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var scanValue string
	// Get the value. Should not loop more than once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&scanValue)
		if err != nil {
			// Unusual, worthy of panic
			panic(err.Error())
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		// Unusual, worthy of panic
		panic(err.Error())
	}
	if counter > 1 {
		panic("Duplicate members in set! " + value)
	}
	return counter > 0, nil
}

// Get all elements of the set
func (s *Set) All() ([]string, error) {
	rows, err := s.file.db.Query("SELECT " + setCol + " FROM " + s.table)
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

// Deprecated, please use .All() instead
func (s *Set) GetAll() ([]string, error) {
	return s.All()
}

// Remove an element from the set
func (s *Set) Del(value string) error {

	// Remove a value from the table
	_, err := s.file.db.Exec("DELETE FROM "+s.table+" WHERE "+setCol+" = ?", value)
	return err
}

// Remove this set
func (s *Set) Remove() error {
	// Remove the table
	_, err := s.file.db.Exec("DROP TABLE " + s.table)
	return err
}

// Clear the list contents
func (s *Set) Clear() error {
	// Clear the table
	_, err := s.file.db.Exec("TRUNCATE TABLE " + s.table)
	return err
}

/* --- HashMap functions --- */

// Create a new hashmap
func NewHashMap(file *File, name string) (*HashMap, error) {
	h := &HashMap{file, name}
	// Using three columns: element id, key and value
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s %s, %s %s, %s %s)", name, ownerCol, defaultStringType, keyCol, defaultStringType, valCol, defaultStringType)
	if _, err := h.file.db.Exec(query); err != nil {
		return nil, err
	}
	if Verbose {
		log.Println("Created table " + name + " in database")
	}
	return h, nil
}

// Set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (h *HashMap) Set(owner, key, value string) error {

	// See if the owner and key already exists
	ok, err := h.Has(owner, key)
	if err != nil {
		return err
	}
	if Verbose {
		log.Printf("%s/%s exists? %v\n", owner, key, ok)
	}
	if ok {
		_, err = h.file.db.Exec("UPDATE "+h.table+" SET "+valCol+" = ? WHERE "+ownerCol+" = ? AND "+keyCol+" = ?", value, owner, key)
		if Verbose {
			log.Println("Updated the table: " + h.table)
		}
	} else {
		_, err = h.file.db.Exec("INSERT INTO "+h.table+" ("+ownerCol+", "+keyCol+", "+valCol+") VALUES (?, ?, ?)", owner, key, value)
		if Verbose {
			log.Println("Added to the table: " + h.table)
		}
	}
	return err
}

// Get a value from a hashmap given the element id (for instance a user id) and the key (for instance "password").
func (h *HashMap) Get(owner, key string) (string, error) {
	rows, err := h.file.db.Query("SELECT "+valCol+" FROM "+h.table+" WHERE "+ownerCol+" = ? AND "+keyCol+" = ?", owner, key)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var value string
	// Get the value. Should only loop once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			// Unusual, worthy of panic
			panic(err.Error())
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		// Unusual, worthy of panic
		panic(err.Error())
	}
	if counter == 0 {
		return "", errors.New("No such owner/key: " + owner + "/" + key)
	}

	return value, nil
}

// Check if a given owner + key is in the hash map
func (h *HashMap) Has(owner, key string) (bool, error) {
	rows, err := h.file.db.Query("SELECT "+valCol+" FROM "+h.table+" WHERE "+ownerCol+" = ? AND "+keyCol+" = ?", owner, key)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var value string
	// Get the value. Should only loop once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			// Unusual, worthy of panic
			panic(err.Error())
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		// Unusual, worthy of panic
		panic(err.Error())
	}
	if counter > 1 {
		panic("Duplicate keys in hash map! " + value)
	}
	return counter > 0, nil
}

// Check if a given owner exists as a hash map at all
func (h *HashMap) Exists(owner string) (bool, error) {
	rows, err := h.file.db.Query("SELECT "+valCol+" FROM "+h.table+" WHERE "+ownerCol+" = ?", owner)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var value string
	// Get the value. Should only loop once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			// Unusual, worthy of panic
			panic(err.Error())
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		// Unusual, worthy of panic
		panic(err.Error())
	}
	return counter > 0, nil
}

// Get all owners (not keys, not values) for all hash elements
func (h *HashMap) All() ([]string, error) {
	rows, err := h.file.db.Query("SELECT " + ownerCol + " FROM " + h.table)
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

// Deprecated, please use .All() instead
func (h *HashMap) GetAll() ([]string, error) {
	return h.All()
}

// Get all keys for a given owner
func (h *HashMap) Keys(owner string) ([]string, error) {
	rows, err := h.file.db.Query("SELECT "+keyCol+" FROM "+h.table+" WHERE "+ownerCol+"= ?", owner)
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

// Remove a key for an entry in a hashmap (for instance the email field for a user)
func (h *HashMap) DelKey(owner, key string) error {
	// Remove a key from the hashmap
	_, err := h.file.db.Exec("DELETE FROM "+h.table+" WHERE "+ownerCol+" = ? AND "+keyCol+" = ?", owner, key)
	return err
}

// Remove an element (for instance a user)
func (h *HashMap) Del(owner string) error {
	// Remove an element id from the table
	results, err := h.file.db.Exec("DELETE FROM "+h.table+" WHERE "+ownerCol+" = ?", owner)
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
	_, err := h.file.db.Exec("DROP TABLE " + h.table)
	return err
}

// Clear the contents
func (h *HashMap) Clear() error {
	// Clear the table
	_, err := h.file.db.Exec("TRUNCATE TABLE " + h.table)
	return err
}

/* --- KeyValue functions --- */

// Create a new key/value
func NewKeyValue(file *File, name string) (*KeyValue, error) {
	kv := &KeyValue{file, name}
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s %s, %s %s)", name, keyCol, defaultStringType, valCol, defaultStringType)
	if _, err := kv.file.db.Exec(query); err != nil {
		return nil, err
	}
	if Verbose {
		log.Println("Created table " + name + " in database")
	}
	return kv, nil

}

// Set a key and value
func (kv *KeyValue) Set(key, value string) error {

	if _, err := kv.Get(key); err != nil {
		// Key does not exist, create it
		_, err = kv.file.db.Exec("INSERT INTO "+kv.table+" ("+keyCol+", "+valCol+") VALUES (?, ?)", key, value)
		return err
	}
	// Key exists, update the value
	_, err := kv.file.db.Exec("UPDATE "+kv.table+" SET "+valCol+" = ? WHERE "+keyCol+" = ?", value, key)
	return err
}

// Get a value given a key
func (kv *KeyValue) Get(key string) (string, error) {
	rows, err := kv.file.db.Query("SELECT "+valCol+" FROM "+kv.table+" WHERE "+keyCol+" = ?", key)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var value string
	// Get the value. Should only loop once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			// Unusual, worthy of panic
			panic(err.Error())
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		// Unusual, worthy of panic
		panic(err.Error())
	}
	if counter != 1 {
		return "", errors.New("Wrong number of keys in KeyValue table: " + kv.table)
	}

	return value, nil
}

// Increase the value of a key, returns the new value
// Returns an empty string if there were errors,
// or "0" if the key does not already exist.
func (kv *KeyValue) Inc(key string) (string, error) {
	// Retreieve the current value, if any
	num := 0
	// See if we can fetch an existing value. NOTE: "== nil"
	if val, err := kv.Get(key); err == nil {
		// See if we can convert the value to a number. NOTE: "== nil"
		if converted, err2 := strconv.Atoi(val); err2 == nil {
			num = converted
		}
	} else {
		// The key does not exist, create a new one.
		// This is to reflect the behavior of INCR in Redis.
		NewKeyValue(kv.file, kv.table)
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
	_, err := kv.file.db.Exec("DELETE FROM "+kv.table+" WHERE "+keyCol+" = ?", key)
	return err
}

// Remove this key/value
func (kv *KeyValue) Remove() error {
	// Remove the table
	_, err := kv.file.db.Exec("DROP TABLE " + kv.table)
	return err
}

// Clear this key/value
func (kv *KeyValue) Clear() error {
	// Remove the table
	_, err := kv.file.db.Exec("TRUNCATE TABLE " + kv.table)
	return err
}
