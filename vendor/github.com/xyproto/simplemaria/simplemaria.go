// Package simplemaria offers a simple way to use a MySQL/MariaDB database.
// This database backend is interchangeable with xyproto/simpleredis and
// xyproto/simplebolt, since they all use xyproto/pinterface.
package simplemaria

import (
	"database/sql"
	"errors"
	"fmt"
	// Use the mysql database driver
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strconv"
	"strings"
)

const (
	// Version number. Stable API within major version numbers.
	Version = 3.2
)

// Host represents a specific database at a database host
type Host struct {
	db      *sql.DB
	dbname  string
	rawUTF8 bool
}

// Common for each of the db datastructures used here
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
	defaultDatabaseServer = ""     // "username:password@server:port/"
	defaultDatabaseName   = "test" // "main"
	defaultStringType     = "TEXT" // "VARCHAR(65535)"
	defaultPort           = 3306

	// Requires MySQL >= 5.53 and MariaDB >= ? for utf8mb4
	charset = "utf8mb4" // "utf8"

	// Column names
	listCol  = "a_list"
	setCol   = "a_set"
	keyCol   = "property"
	valCol   = "value"
	ownerCol = "owner"
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
	db, err := sql.Open("mysql", newConnectionString)
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

// Test if a given database server is up and running.
func TestConnectionHostWithDSN(connectionString string) (err error) {
	// Connect to the given host:port
	db, err := sql.Open("mysql", connectionString)
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

/* --- Host functions --- */

// Create a new database connection.
// connectionString may be on the form "username:password@host:port/database".
func NewHost(connectionString string) *Host {

	newConnectionString, dbname := rebuildConnectionString(connectionString)

	db, err := sql.Open("mysql", newConnectionString)
	if err != nil {
		log.Fatalln("Could not connect to " + newConnectionString + "!")
	}
	host := &Host{db, dbname, false}
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

// Create a new database connection with a valid DSN.
func NewHostWithDSN(connectionString string, dbname string) *Host {

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatalln("Could not connect to " + connectionString + "!")
	}
	host := &Host{db, dbname, false}
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

// The default database connection
func New() *Host {
	connectionString := defaultDatabaseServer + defaultDatabaseName
	if !strings.HasSuffix(defaultDatabaseServer, "/") {
		connectionString = defaultDatabaseServer + "/" + defaultDatabaseName
	}
	return NewHost(connectionString)
}

// Should the UTF-8 data be raw, and not hex encoded and compressed?
func (host *Host) SetRawUTF8(enabled bool) {
	host.rawUTF8 = enabled
}

// Select a different database. Create the database if needed.
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
	if _, err := host.db.Exec("CREATE DATABASE IF NOT EXISTS " + host.dbname + " CHARACTER SET = " + charset); err != nil {
		return err
	}
	if Verbose {
		log.Println("Created database " + host.dbname)
	}
	return nil
}

// Use the host.dbname database
func (host *Host) useDatabase() error {
	if _, err := host.db.Exec("USE " + host.dbname); err != nil {
		return err
	}
	if Verbose {
		log.Println("Using database " + host.dbname)
	}
	return nil
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
	l := &List{host, name}
	if _, err := l.host.db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (id INT PRIMARY KEY AUTO_INCREMENT, " + listCol + " " + defaultStringType + ")"); err != nil {
		return nil, err
	}
	if Verbose {
		log.Println("Created table " + name + " in database " + host.dbname)
	}
	return l, nil
}

// Add an element to the list
func (l *List) Add(value string) error {
	if !l.host.rawUTF8 {
		Encode(&value)
	}
	_, err := l.host.db.Exec("INSERT INTO "+l.table+" ("+listCol+") VALUES (?)", value)
	return err
}

// Get all elements of a list
func (l *List) All() ([]string, error) {
	rows, err := l.host.db.Query("SELECT " + listCol + " FROM " + l.table + " ORDER BY id")
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
		if !l.host.rawUTF8 {
			Decode(&value)
		}
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
	rows, err := l.host.db.Query("SELECT " + listCol + " FROM " + l.table + " WHERE id = (SELECT MAX(id) FROM " + l.table + ")")
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
	if !l.host.rawUTF8 {
		Decode(&value)
	}
	return value, nil
}

// Deprecated, please use .Last() instead
func (l *List) GetLast() (string, error) {
	return l.Last()
}

// Get the last N elements of a list
func (l *List) LastN(n int) ([]string, error) {
	rows, err := l.host.db.Query("SELECT " + listCol + " FROM (SELECT * FROM " + l.table + " ORDER BY id DESC limit " + strconv.Itoa(n) + ")sub ORDER BY id ASC")
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
		if !l.host.rawUTF8 {
			Decode(&value)
		}
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
	_, err := l.host.db.Exec("DROP TABLE " + l.table)
	return err
}

// Clear the list contents
func (l *List) Clear() error {
	// Clear the table
	_, err := l.host.db.Exec("TRUNCATE TABLE " + l.table)
	return err
}

/* --- Set functions --- */

// Create a new set
func NewSet(host *Host, name string) (*Set, error) {
	s := &Set{host, name}
	if _, err := s.host.db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (" + setCol + " " + defaultStringType + ")"); err != nil {
		return nil, err
	}
	if Verbose {
		log.Println("Created table " + name + " in database " + host.dbname)
	}
	return s, nil
}

// Add an element to the set
func (s *Set) Add(value string) error {
	originalValue := value
	if !s.host.rawUTF8 {
		Encode(&value)
	}
	// Check if the value is not already there before adding
	has, err := s.Has(originalValue)
	if !has && (err == nil) {
		_, err = s.host.db.Exec("INSERT INTO "+s.table+" ("+setCol+") VALUES (?)", value)
	}
	return err
}

// Check if a given value is in the set
func (s *Set) Has(value string) (bool, error) {
	if !s.host.rawUTF8 {
		Encode(&value)
	}
	rows, err := s.host.db.Query("SELECT "+setCol+" FROM "+s.table+" WHERE "+setCol+" = ?", value)
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
	rows, err := s.host.db.Query("SELECT " + setCol + " FROM " + s.table)
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
		if !s.host.rawUTF8 {
			Decode(&value)
		}
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
	if !s.host.rawUTF8 {
		Encode(&value)
	}
	// Remove a value from the table
	_, err := s.host.db.Exec("DELETE FROM "+s.table+" WHERE "+setCol+" = ?", value)
	return err
}

// Remove this set
func (s *Set) Remove() error {
	// Remove the table
	_, err := s.host.db.Exec("DROP TABLE " + s.table)
	return err
}

// Clear the list contents
func (s *Set) Clear() error {
	// Clear the table
	_, err := s.host.db.Exec("TRUNCATE TABLE " + s.table)
	return err
}

/* --- HashMap functions --- */

// Create a new hashmap
func NewHashMap(host *Host, name string) (*HashMap, error) {
	h := &HashMap{host, name}
	// Using three columns: element id, key and value
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s %s, %s %s, %s %s)", name, ownerCol, defaultStringType, keyCol, defaultStringType, valCol, defaultStringType)
	if _, err := h.host.db.Exec(query); err != nil {
		return nil, err
	}
	if Verbose {
		log.Println("Created table " + name + " in database " + host.dbname)
	}
	return h, nil
}

// Set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (h *HashMap) Set(owner, key, value string) error {
	if !h.host.rawUTF8 {
		Encode(&value)
	}
	// See if the owner and key already exists
	ok, err := h.Has(owner, key)
	if err != nil {
		return err
	}
	if Verbose {
		log.Printf("%s/%s exists? %v\n", owner, key, ok)
	}
	if ok {
		_, err = h.host.db.Exec("UPDATE "+h.table+" SET "+valCol+" = ? WHERE "+ownerCol+" = ? AND "+keyCol+" = ?", value, owner, key)
		if Verbose {
			log.Println("Updated the table: " + h.table)
		}
	} else {
		_, err = h.host.db.Exec("INSERT INTO "+h.table+" ("+ownerCol+", "+keyCol+", "+valCol+") VALUES (?, ?, ?)", owner, key, value)
		if Verbose {
			log.Println("Added to the table: " + h.table)
		}
	}
	return err
}

// Get a value from a hashmap given the element id (for instance a user id) and the key (for instance "password").
func (h *HashMap) Get(owner, key string) (string, error) {
	rows, err := h.host.db.Query("SELECT "+valCol+" FROM "+h.table+" WHERE "+ownerCol+" = ? AND "+keyCol+" = ?", owner, key)
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
	if !h.host.rawUTF8 {
		Decode(&value)
	}
	return value, nil
}

// Check if a given owner + key is in the hash map
func (h *HashMap) Has(owner, key string) (bool, error) {
	rows, err := h.host.db.Query("SELECT "+valCol+" FROM "+h.table+" WHERE "+ownerCol+" = ? AND "+keyCol+" = ?", owner, key)
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
	rows, err := h.host.db.Query("SELECT "+valCol+" FROM "+h.table+" WHERE "+ownerCol+" = ?", owner)
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
	rows, err := h.host.db.Query("SELECT " + ownerCol + " FROM " + h.table)
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
	rows, err := h.host.db.Query("SELECT "+keyCol+" FROM "+h.table+" WHERE "+ownerCol+"= ?", owner)
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
	_, err := h.host.db.Exec("DELETE FROM "+h.table+" WHERE "+ownerCol+" = ? AND "+keyCol+" = ?", owner, key)
	return err
}

// Remove an element (for instance a user)
func (h *HashMap) Del(owner string) error {
	// Remove an element id from the table
	results, err := h.host.db.Exec("DELETE FROM "+h.table+" WHERE "+ownerCol+" = ?", owner)
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
	_, err := h.host.db.Exec("DROP TABLE " + h.table)
	return err
}

// Clear the contents
func (h *HashMap) Clear() error {
	// Clear the table
	_, err := h.host.db.Exec("TRUNCATE TABLE " + h.table)
	return err
}

/* --- KeyValue functions --- */

// Create a new key/value
func NewKeyValue(host *Host, name string) (*KeyValue, error) {
	kv := &KeyValue{host, name}
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s %s, %s %s)", name, keyCol, defaultStringType, valCol, defaultStringType)
	if _, err := kv.host.db.Exec(query); err != nil {
		return nil, err
	}
	if Verbose {
		log.Println("Created table " + name + " in database " + host.dbname)
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
		_, err = kv.host.db.Exec("INSERT INTO "+kv.table+" ("+keyCol+", "+valCol+") VALUES (?, ?)", key, value)
		return err
	}
	// Key exists, update the value
	_, err := kv.host.db.Exec("UPDATE "+kv.table+" SET "+valCol+" = ? WHERE "+keyCol+" = ?", value, key)
	return err
}

// Get a value given a key
func (kv *KeyValue) Get(key string) (string, error) {
	rows, err := kv.host.db.Query("SELECT "+valCol+" FROM "+kv.table+" WHERE "+keyCol+" = ?", key)
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
	if !kv.host.rawUTF8 {
		Decode(&value)
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
	_, err := kv.host.db.Exec("DELETE FROM "+kv.table+" WHERE "+keyCol+" = ?", key)
	return err
}

// Remove this key/value
func (kv *KeyValue) Remove() error {
	// Remove the table
	_, err := kv.host.db.Exec("DROP TABLE " + kv.table)
	return err
}

// Clear this key/value
func (kv *KeyValue) Clear() error {
	// Remove the table
	_, err := kv.host.db.Exec("TRUNCATE TABLE " + kv.table)
	return err
}
