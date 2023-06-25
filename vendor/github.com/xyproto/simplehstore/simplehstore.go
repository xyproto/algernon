// Package simplehstore offers a simple way to use a PostgreSQL database with HSTORE.
// The database back end is interchangeable with Redis (xyproto/simpleredis), BoltDB (xyproto/simplebolt) and
// MariaDB/MySQL (xyproto/simplemaria) by using the interfaces in the xyproto/pinterface package.
package simplehstore

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	// Using the PostgreSQL database engine
	pq "github.com/lib/pq"
	"github.com/xyproto/env/v2"
)

const (
	// VersionString is the current version of simplehstore.
	VersionString = "1.8.1"

	defaultStringType = "TEXT"
	defaultPort       = 5432
	encoding          = "UTF8"
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

var defaultConnectionString = func() string {
	password := env.Str("POSTGRES_PASSWORD")
	s := env.Str("POSTGRES_USER", "postgres")
	if password != "" {
		s += ":" + password
	}
	s += "@127.0.0.1/" // for CI testing
	return s
}()

var (
	// The default "username:password@host:port/database" that the database is running at
	defaultDatabaseName = env.Str("POSTGRES_DB", "postgres")

	// ErrNoAvailableValues is used as an error if an SQL query returns no values
	ErrNoAvailableValues = errors.New("no available values")
	// ErrTooFewResults is used as an error if an SQL query returns too few results
	ErrTooFewResults = errors.New("too few results")

	// Column names
	listCol  = "a_list"
	setCol   = "a_set"
	ownerCol = "owner"
	kvPrefix = "a_kv_"
)

// SetColumnNames can be used to change the column names and prefixes that are used in the PostgreSQL tables.
// The default values are: "a_list", "a_set", "owner" and "a_kv_".
func SetColumnNames(list, set, hashMapOwner, keyValuePrefix string) {
	listCol = list
	setCol = set
	ownerCol = hashMapOwner
	kvPrefix = keyValuePrefix
}

// TestConnection checks if the local database server is up and running
func TestConnection() (err error) {
	return TestConnectionHost(defaultConnectionString)
}

// TestConnectionHost checks if a given database server is up and running.
// connectionString may be on the form "username:password@host:port/database".
// The database name is ignored.
func TestConnectionHost(connectionString string) error {
	newConnectionString, _ := rebuildConnectionString(connectionString, false)
	// Connect to the given host:port
	db, err := sql.Open("postgres", newConnectionString)
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

// TestConnectionHostWithDSN checks if a given database server is up and running.
func TestConnectionHostWithDSN(connectionString string) error {
	// Connect to the given host:port
	db, err := sql.Open("postgres", connectionString)
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

// NewHost sets up a new database connection.
// connectionString may be on the form "username:password@host:port/database".
func NewHost(connectionString string) *Host {
	host, err := NewHost2(connectionString)
	if err != nil {
		log.Fatalln(err)
	}
	return host
}

// NewHost2 sets up a new database connection.
// connectionString may be on the form "username:password@host:port/database".
// An error may be returned.
func NewHost2(connectionString string) (*Host, error) {
	newConnectionString, dbname := rebuildConnectionString(connectionString, true)
	db, err := sql.Open("postgres", newConnectionString)
	if err != nil {
		return nil, fmt.Errorf("could not connect to %s", newConnectionString)
	}
	host := &Host{db, pq.QuoteIdentifier(dbname), false}
	if err := host.Ping(); err != nil {
		return nil, fmt.Errorf("database host does not reply to ping: %s", err)
	}
	if err := host.createDatabase(); err != nil {
		return nil, fmt.Errorf("could not create database %s: %s", host.dbname, err)
	}
	if err := host.useDatabase(); err != nil {
		return nil, fmt.Errorf("could not use database %s: %s", host.dbname, err)
	}
	return host, nil
}

// NewHostWithDSN creates a new database connection with a valid DSN.
func NewHostWithDSN(connectionString string, dbname string) *Host {
	host, err := NewHostWithDSN2(connectionString, dbname)
	if err != nil {
		log.Fatalln(err)
	}
	return host
}

// NewHostWithDSN2 creates a new database connection with a valid DSN.
// An error may be returned.
func NewHostWithDSN2(connectionString string, dbname string) (*Host, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("could not connect to %s", connectionString)
	}
	host := &Host{db, pq.QuoteIdentifier(dbname), false}
	if err := host.Ping(); err != nil {
		return nil, fmt.Errorf("database host does not reply to ping: %s", err)
	}
	if err := host.createDatabase(); err != nil {
		return nil, fmt.Errorf("could not create database %s: %s", host.dbname, err)
	}
	if err := host.useDatabase(); err != nil {
		return nil, fmt.Errorf("could not use database %s: %s", host.dbname, err)
	}
	return host, nil
}

// New sets up a connection to the default (local) database host
func New() *Host {
	connectionString := defaultConnectionString + defaultDatabaseName
	if !strings.HasSuffix(defaultConnectionString, "/") {
		connectionString = defaultConnectionString + "/" + defaultDatabaseName
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
	return host.useDatabase()
}

// Will create the database if it does not already exist
func (host *Host) createDatabase() error {
	if _, err := host.db.Exec(fmt.Sprintf("CREATE DATABASE %s WITH ENCODING '%s'", host.dbname, encoding)); err != nil {
		if !strings.HasSuffix(err.Error(), "already exists") {
			return err
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

// Database returns the underlying *sql.DB database struct
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
