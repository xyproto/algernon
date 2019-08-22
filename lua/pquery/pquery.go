// Package pquery provides Lua functions for storing Lua functions in a database
package pquery

import (
	"database/sql"
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/gopher-lua"
	"github.com/xyproto/pinterface"
	"strings"

	// Using the PostgreSQL database engine
	_ "github.com/lib/pq"
)

// Library class is for storing and loading Lua source code to and from a data structure.

const (
	defaultQuery            = "SELECT version()"
	defaultConnectionString = "host=localhost port=5432 user=postgres dbname=test sslmode=disable"
)

// Load makes functions related to building a library of Lua code available
func Load(L *lua.LState, perm pinterface.IPermissions) {

	// Register the PQ function
	L.SetGlobal("PQ", L.NewFunction(func(L *lua.LState) int {

		// Check if the optional argument is given
		query := defaultQuery
		if L.GetTop() == 1 {
			query = L.ToString(1)
			if query == "" {
				query = defaultQuery
			}
		}
		connectionString := defaultConnectionString
		if L.GetTop() == 2 {
			connectionString = L.ToString(2)
		}

		db, err := sql.Open("postgres", connectionString)
		if err != nil {
			log.Error("Could not connect to database using " + connectionString + ": " + err.Error())
			return 0 // No results
		}
		//log.Info(fmt.Sprintf("PostgreSQL database: %v (%T)\n", db, db))
		rows, err := db.Query(query)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, ": connect: connection refused") {
				log.Info("PostgreSQL connection string: " + connectionString)
				log.Info("PostgreSQL query: " + query)
				log.Error("Could not connect to database: " + errMsg)
			} else if strings.Contains(errMsg, "missing") && strings.Contains(errMsg, "in connection info string") {
				log.Info("PostgreSQL connection string: " + connectionString)
				log.Info("PostgreSQL query: " + query)
				log.Error(errMsg)
			} else {
				log.Info("PostgreSQL query: " + query)
				log.Error("Query failed: " + errMsg)
			}
			return 0 // No results
		}
		if rows == nil {
			// Return an empty table
			table := convert.Strings2table(L, []string{})
			L.Push(table)
			return 1 // number of results
		}
		// Return the rows as a table
		var (
			values []string
			value  string
		)
		for rows.Next() {
			err = rows.Scan(&value)
			if err != nil {
				break
			}
			values = append(values, value)
		}
		// Convert the strings to a Lua table
		table := convert.Strings2table(L, values)
		// Return the table
		L.Push(table)
		return 1 // number of results
	}))

}
