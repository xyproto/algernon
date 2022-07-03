package mssql

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/lua/convert"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/pinterface"

	// Using the MSSQL database engine
	_ "github.com/denisenkom/go-mssqldb"
)

const (
	defaultQuery            = "SELECT @@VERSION"
	defaultConnectionString = "server=localhost;user=sa;password=Password123,port=1433"
)

var (
	// global map from connection string to database connection, to reuse connections, protected by a mutex
	reuseDB  = make(map[string]*sql.DB)
	reuseMut = &sync.RWMutex{}
)

// LValueWrapper decorates lua.LValue to help retieve values from the database.
type LValueWrapper struct {
	LValue lua.LValue
}

// Scan implements the sql.Scanner interface for database deserialization.
func (w *LValueWrapper) Scan(value interface{}) error {

	if value == nil {
		*w = LValueWrapper{lua.LNil}
		return nil
	}

	switch v := value.(type) {

	case float32:
		*w = LValueWrapper{lua.LNumber(float64(v))}

	case float64:
		*w = LValueWrapper{lua.LNumber(v)}

	case int64:
		*w = LValueWrapper{lua.LNumber(float64(v))}

	case string:
		*w = LValueWrapper{lua.LString(v)}

	case []byte:
		*w = LValueWrapper{lua.LString(string(v))}

	case time.Time:
		*w = LValueWrapper{lua.LNumber(float64(v.Unix()))}

	default:
		return fmt.Errorf("unable to scan type %T into lua value wrapper", value)

	}

	return nil
}

// LValueWrappers is a convenience type to easily map to a slice of lua.LValue
type LValueWrappers []LValueWrapper

// Unwrap produces a slice of lua.LValue from the given LValueWrappers
func (w LValueWrappers) Unwrap() (s []lua.LValue) {
	s = make([]lua.LValue, len(w))
	for i, v := range w {
		s[i] = v.LValue
	}
	return
}

// Interfaces returns a slice of interface{} values from the given LValueWrappers
func (w LValueWrappers) Interfaces() (s []interface{}) {
	s = make([]interface{}, len(w))
	for i := range w {
		s[i] = &w[i]
	}
	return
}

// Load makes functions related to building a library of Lua code available
func Load(L *lua.LState, perm pinterface.IPermissions) {

	// Register the MSSQL function
	L.SetGlobal("MSSQL", L.NewFunction(func(L *lua.LState) int {

		// Check if the optional argument is given
		query := defaultQuery
		if L.GetTop() >= 1 {
			query = L.ToString(1)
			if query == "" {
				query = defaultQuery
			}
		}
		connectionString := defaultConnectionString
		if L.GetTop() >= 2 {
			connectionString = L.ToString(2)
		}

		// Get arguments
		var queryArgs []interface{}
		if L.GetTop() >= 3 {
			args := L.ToTable(3)
			args.ForEach(func(k lua.LValue, v lua.LValue) {
				switch k.Type() {
				case lua.LTNumber:
					queryArgs = append(queryArgs, v.String())
				case lua.LTString:
					queryArgs = append(queryArgs, sql.Named(k.String(), v.String()))
				}
			})
		}

		// Check if there is a connection that can be reused
		var db *sql.DB = nil
		reuseMut.RLock()
		conn, ok := reuseDB[connectionString]
		reuseMut.RUnlock()

		if ok {
			// It exists, but is it still alive?
			err := conn.Ping()
			if err != nil {
				// no
				//log.Info("did not reuse the connection")
				reuseMut.Lock()
				delete(reuseDB, connectionString)
				reuseMut.Unlock()
			} else {
				// yes
				//log.Info("reused the connection")
				db = conn
			}
		}
		// Create a new connection, if needed
		var err error
		if db == nil {
			db, err = sql.Open("sqlserver", connectionString)
			if err != nil {
				log.Error("Could not connect to database using " + connectionString + ": " + err.Error())
				return 0 // No results
			}
			// Save the connection for later
			reuseMut.Lock()
			reuseDB[connectionString] = db
			reuseMut.Unlock()
		}
		//log.Info(fmt.Sprintf("MSSQL database: %v (%T)\n", db, db))
		reuseMut.Lock()
		rows, err := db.Query(query, queryArgs...)
		reuseMut.Unlock()
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, ": connect: connection refused") {
				log.Info("MSSQL connection string: " + connectionString)
				log.Info("MSSQL query: " + query)
				log.Error("Could not connect to database: " + errMsg)
			} else if strings.Contains(errMsg, "missing") && strings.Contains(errMsg, "in connection info string") {
				log.Info("MSSQL connection string: " + connectionString)
				log.Info("MSSQL query: " + query)
				log.Error(errMsg)
			} else {
				log.Info("MSSQL query: " + query)
				log.Error("Query failed: " + errMsg)
			}
			return 0 // No results
		}
		if rows == nil {
			// Return an empty table
			L.Push(L.NewTable())
			return 1 // number of results
		}
		cols, err := rows.Columns()
		if err != nil {
			log.Error("Failed to get columns: " + err.Error())
			return 0
		}
		// Return the rows as a 2-dimensional table
		// Outer table is an array of rows
		// Inner tables are maps of values with column names as keys
		var (
			m      map[string]lua.LValue
			maps   []map[string]lua.LValue
			values LValueWrappers
			cname  string
		)
		for rows.Next() {
			values = make(LValueWrappers, len(cols))
			err = rows.Scan(values.Interfaces()...)
			if err != nil {
				log.Error("Failed to scan data: " + err.Error())
				break
			}
			m = make(map[string]lua.LValue, len(cols))
			for i, v := range values.Unwrap() {
				cname = cols[i]
				m[cname] = v
			}
			maps = append(maps, m)
		}
		// Convert the strings to a Lua table
		table := convert.LValueMaps2table(L, maps)
		// Return the table
		L.Push(table)
		return 1 // number of results
	}))

}
