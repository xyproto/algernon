// Package sqlite provides Lua functions for querying SQLite databases
package sqlite

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/lua/convert"
	lua "github.com/xyproto/gopher-lua"
)

const (
	defaultQuery    = "SELECT sqlite_version()"
	defaultFilename = "sqlite.db"

	// Class identifier for the SQLiteFile userdata in Lua
	lSQLiteFileClass = "SQLITEFILE"
)

// validIdentifier matches safe SQLite identifiers (letters, digits, underscores)
var validIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// sqliteFileData holds a database connection and an optional active transaction
type sqliteFileData struct {
	db       *sql.DB
	tx       *sql.Tx
	filename string
}

// queryable returns the active transaction if set, otherwise the database connection
func (d *sqliteFileData) queryable() queryExecer {
	if d.tx != nil {
		return d.tx
	}
	return d.db
}

// queryExecer is the common interface between *sql.DB and *sql.Tx
type queryExecer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}

var (
	// global map from filename to database connection, to reuse connections, protected by a mutex
	reuseDB  = make(map[string]*sql.DB)
	reuseMut = &sync.RWMutex{}
)

// getDB returns a reused or new database connection for the given filename
func getDB(filename string) (*sql.DB, error) {
	reuseMut.RLock()
	conn, ok := reuseDB[filename]
	reuseMut.RUnlock()

	if ok {
		// It exists, but is it still alive?
		err := conn.Ping()
		if err != nil {
			reuseMut.Lock()
			delete(reuseDB, filename)
			reuseMut.Unlock()
		} else {
			return conn, nil
		}
	}

	// Create a new connection
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}

	// Save the connection for later
	reuseMut.Lock()
	reuseDB[filename] = db
	reuseMut.Unlock()

	return db, nil
}

// queryRows executes a query and returns the results as a Lua table.
// Each row is a table with column names as keys.
func queryRows(L *lua.LState, qe queryExecer, query string, args ...any) *lua.LTable {
	rows, err := qe.Query(query, args...)
	if err != nil {
		logrus.Info("SQLite query: " + query)
		logrus.Error("Query failed: " + err.Error())
		return L.NewTable()
	}
	if rows == nil {
		return L.NewTable()
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		logrus.Error("Failed to get columns: " + err.Error())
		return L.NewTable()
	}

	var (
		m      map[string]lua.LValue
		maps   []map[string]lua.LValue
		values convert.LValueWrappers
		cname  string
	)
	for rows.Next() {
		values = make(convert.LValueWrappers, len(cols))
		err = rows.Scan(values.Interfaces()...)
		if err != nil {
			logrus.Error("Failed to scan data: " + err.Error())
			break
		}
		m = make(map[string]lua.LValue, len(cols))
		for i, v := range values.Unwrap() {
			cname = cols[i]
			m[cname] = v
		}
		maps = append(maps, m)
	}

	return convert.LValueMaps2table(L, maps)
}

// ensureCollection creates the backing table for a JSON document collection if needed.
// The collection name must be a valid SQLite identifier.
func ensureCollection(qe queryExecer, collection string) error {
	if !validIdentifier.MatchString(collection) {
		return fmt.Errorf("invalid collection name: %s", collection)
	}
	_, err := qe.Exec("CREATE TABLE IF NOT EXISTS " + collection + " (id INTEGER PRIMARY KEY AUTOINCREMENT, data JSON)")
	return err
}

// buildWhereClause builds a WHERE clause from a Lua table of field=value pairs,
// using json_extract for matching against JSON document fields.
// Returns the clause string and the parameter values.
func buildWhereClause(L *lua.LState, table *lua.LTable) (string, []any) {
	var conditions []string
	var params []any
	table.ForEach(func(k lua.LValue, v lua.LValue) {
		key := k.String()
		if !validIdentifier.MatchString(key) {
			return
		}
		conditions = append(conditions, "json_extract(data, '$."+key+"') = ?")
		switch val := v.(type) {
		case lua.LNumber:
			params = append(params, float64(val))
		case lua.LString:
			params = append(params, string(val))
		case lua.LBool:
			params = append(params, bool(val))
		default:
			params = append(params, v.String())
		}
	})
	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), params
}

// --- SQLiteFile userdata methods ---

// Get the first argument, "self", and cast it from userdata to a *sqliteFileData
func checkSQLiteFile(L *lua.LState) *sqliteFileData {
	ud := L.CheckUserData(1)
	if data, ok := ud.Value.(*sqliteFileData); ok {
		return data
	}
	L.ArgError(1, "SQLiteFile expected")
	return nil
}

// String representation
// Returns the database filename
// tostring(db) -> string
func sqlfToString(L *lua.LState) int {
	data := checkSQLiteFile(L)
	L.Push(lua.LString(data.filename))
	return 1 // Number of returned values
}

// Execute a SQL query and return the results as a table.
// db:query(string, [table]) -> table
func sqlfQuery(L *lua.LState) int {
	data := checkSQLiteFile(L) // arg 1
	query := L.ToString(2)     // arg 2
	if query == "" {
		query = defaultQuery
	}
	var args []any
	if L.GetTop() >= 3 {
		args = convert.ExtractParams(L, 3)
	}
	L.Push(queryRows(L, data.queryable(), query, args...))
	return 1 // Number of returned values
}

// Execute a SQL statement and return the number of affected rows and success.
// db:exec(string, [table]) -> number, bool
func sqlfExec(L *lua.LState) int {
	data := checkSQLiteFile(L) // arg 1
	query := L.ToString(2)     // arg 2
	if query == "" {
		L.Push(lua.LNumber(0))
		L.Push(lua.LBool(false))
		return 2 // Number of returned values
	}
	var args []any
	if L.GetTop() >= 3 {
		args = convert.ExtractParams(L, 3)
	}
	result, err := data.queryable().Exec(query, args...)
	if err != nil {
		logrus.Info("SQLite exec: " + query)
		logrus.Error("Exec failed: " + err.Error())
		L.Push(lua.LNumber(0))
		L.Push(lua.LBool(false))
		return 2 // Number of returned values
	}
	affected, err := result.RowsAffected()
	if err != nil {
		affected = 0
	}
	L.Push(lua.LNumber(float64(affected)))
	L.Push(lua.LBool(true))
	return 2 // Number of returned values
}

// Execute a function within a transaction.
// Calls BEGIN, runs the function, then COMMIT on success or ROLLBACK on error.
// db:transaction(function) -> bool
func sqlfTransaction(L *lua.LState) int {
	data := checkSQLiteFile(L) // arg 1
	fn := L.CheckFunction(2)   // arg 2
	tx, err := data.db.Begin()
	if err != nil {
		logrus.Error("Failed to begin transaction: " + err.Error())
		L.Push(lua.LBool(false))
		return 1 // Number of returned values
	}
	// Set the active transaction so that query/exec use it
	data.tx = tx
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}); err != nil {
		data.tx = nil
		if rbErr := tx.Rollback(); rbErr != nil {
			logrus.Error("Failed to rollback: " + rbErr.Error())
		}
		logrus.Error("Transaction failed: " + err.Error())
		L.Push(lua.LBool(false))
		return 1 // Number of returned values
	}
	data.tx = nil
	if err := tx.Commit(); err != nil {
		logrus.Error("Failed to commit: " + err.Error())
		L.Push(lua.LBool(false))
		return 1 // Number of returned values
	}
	L.Push(lua.LBool(true))
	return 1 // Number of returned values
}

// Add a JSON document to a collection. Returns the row ID as a string.
// db:add(string, table) -> string
func sqlfAdd(L *lua.LState) int {
	data := checkSQLiteFile(L)     // arg 1
	collection := L.CheckString(2) // arg 2
	table := L.CheckTable(3)       // arg 3

	if err := ensureCollection(data.queryable(), collection); err != nil {
		logrus.Error("Failed to create collection: " + err.Error())
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}

	jsonData, err := convert.Table2JSON(L, table)
	if err != nil {
		logrus.Error("Failed to convert to JSON: " + err.Error())
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}

	result, err := data.queryable().Exec("INSERT INTO "+collection+" (data) VALUES (?)", jsonData)
	if err != nil {
		logrus.Error("Failed to insert document: " + err.Error())
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}

	id, err := result.LastInsertId()
	if err != nil {
		logrus.Error("Failed to get last insert ID: " + err.Error())
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}

	L.Push(lua.LString(fmt.Sprintf("%d", id)))
	return 1 // Number of returned values
}

// Retrieve a single document from a collection by its row ID.
// db:get(string, string) -> table
func sqlfGet(L *lua.LState) int {
	data := checkSQLiteFile(L)     // arg 1
	collection := L.CheckString(2) // arg 2
	docID := L.CheckString(3)      // arg 3

	if !validIdentifier.MatchString(collection) {
		L.Push(L.NewTable())
		return 1 // Number of returned values
	}

	var docData string
	err := data.queryable().QueryRow("SELECT data FROM "+collection+" WHERE id = ?", docID).Scan(&docData)
	if err != nil {
		L.Push(L.NewTable())
		return 1 // Number of returned values
	}

	L.Push(convert.JSON2table(L, docData))
	return 1 // Number of returned values
}

// Retrieve documents from a collection. Optional filter table for field matching.
// db:docs(string, [table]) -> table
func sqlfDocs(L *lua.LState) int {
	data := checkSQLiteFile(L)     // arg 1
	collection := L.CheckString(2) // arg 2

	if err := ensureCollection(data.queryable(), collection); err != nil {
		logrus.Error("Failed to create collection: " + err.Error())
		L.Push(L.NewTable())
		return 1 // Number of returned values
	}

	query := "SELECT data FROM " + collection
	var params []any

	// Check for an optional filter table
	if L.GetTop() >= 3 {
		filterTable := L.ToTable(3)
		if filterTable != nil {
			where, whereParams := buildWhereClause(L, filterTable)
			query += where
			params = whereParams
		}
	}

	rows, err := data.queryable().Query(query, params...)
	if err != nil {
		logrus.Info("SQLite docs query: " + query)
		logrus.Error("Query failed: " + err.Error())
		L.Push(L.NewTable())
		return 1 // Number of returned values
	}
	defer rows.Close()

	results := L.NewTable()
	var docData string
	for rows.Next() {
		if err := rows.Scan(&docData); err != nil {
			logrus.Error("Failed to scan data: " + err.Error())
			break
		}
		results.Append(convert.JSON2table(L, docData))
	}

	L.Push(results)
	return 1 // Number of returned values
}

// Delete documents from a collection matching the given filter.
// db:del(string, table) -> bool
func sqlfDel(L *lua.LState) int {
	data := checkSQLiteFile(L)     // arg 1
	collection := L.CheckString(2) // arg 2
	filterTable := L.CheckTable(3) // arg 3

	if !validIdentifier.MatchString(collection) {
		L.Push(lua.LBool(false))
		return 1 // Number of returned values
	}

	where, params := buildWhereClause(L, filterTable)
	if where == "" {
		L.Push(lua.LBool(false))
		return 1 // Number of returned values
	}

	_, err := data.queryable().Exec("DELETE FROM "+collection+where, params...)
	if err != nil {
		logrus.Error("Failed to delete documents: " + err.Error())
		L.Push(lua.LBool(false))
		return 1 // Number of returned values
	}

	L.Push(lua.LBool(true))
	return 1 // Number of returned values
}

// Update documents in a collection matching the where-filter, setting fields from the set-table.
// db:update(string, table, table) -> bool
func sqlfUpdate(L *lua.LState) int {
	data := checkSQLiteFile(L)     // arg 1
	collection := L.CheckString(2) // arg 2
	whereTable := L.CheckTable(3)  // arg 3
	setTable := L.CheckTable(4)    // arg 4

	if !validIdentifier.MatchString(collection) {
		L.Push(lua.LBool(false))
		return 1 // Number of returned values
	}

	// Build SET clause using json_set: json_set(data, '$.key1', ?, '$.key2', ?)
	var pairs []string
	var setParams []any
	setTable.ForEach(func(k lua.LValue, v lua.LValue) {
		key := k.String()
		if !validIdentifier.MatchString(key) {
			return
		}
		pairs = append(pairs, "'$."+key+"', ?")
		switch val := v.(type) {
		case lua.LNumber:
			setParams = append(setParams, float64(val))
		case lua.LString:
			setParams = append(setParams, string(val))
		case lua.LBool:
			setParams = append(setParams, bool(val))
		default:
			setParams = append(setParams, v.String())
		}
	})
	if len(pairs) == 0 {
		L.Push(lua.LBool(false))
		return 1 // Number of returned values
	}

	setExpr := "data = json_set(data, " + strings.Join(pairs, ", ") + ")"

	where, whereParams := buildWhereClause(L, whereTable)

	allParams := append(setParams, whereParams...)
	_, err := data.queryable().Exec("UPDATE "+collection+" SET "+setExpr+where, allParams...)
	if err != nil {
		logrus.Error("Failed to update documents: " + err.Error())
		L.Push(lua.LBool(false))
		return 1 // Number of returned values
	}

	L.Push(lua.LBool(true))
	return 1 // Number of returned values
}

// Return the number of documents in a collection.
// db:len(string) -> number
func sqlfLen(L *lua.LState) int {
	data := checkSQLiteFile(L)     // arg 1
	collection := L.CheckString(2) // arg 2

	if !validIdentifier.MatchString(collection) {
		L.Push(lua.LNumber(0))
		return 1 // Number of returned values
	}

	var count int64
	err := data.queryable().QueryRow("SELECT COUNT(*) FROM " + collection).Scan(&count)
	if err != nil {
		L.Push(lua.LNumber(0))
		return 1 // Number of returned values
	}

	L.Push(lua.LNumber(float64(count)))
	return 1 // Number of returned values
}

// Close the database connection. Returns true if successful.
// db:close() -> bool
func sqlfClose(L *lua.LState) int {
	data := checkSQLiteFile(L) // arg 1

	// Evict from the connection pool
	reuseMut.Lock()
	for filename, conn := range reuseDB {
		if conn == data.db {
			delete(reuseDB, filename)
			break
		}
	}
	reuseMut.Unlock()

	L.Push(lua.LBool(nil == data.db.Close()))
	return 1 // Number of returned values
}

// The SQLiteFile methods that are to be registered
var sqlfMethods = map[string]lua.LGFunction{
	"__tostring":  sqlfToString,
	"query":       sqlfQuery,
	"exec":        sqlfExec,
	"transaction": sqlfTransaction,
	"add":         sqlfAdd,
	"get":         sqlfGet,
	"docs":        sqlfDocs,
	"del":         sqlfDel,
	"update":      sqlfUpdate,
	"len":         sqlfLen,
	"close":       sqlfClose,
}

// Load makes functions related to SQLite databases available to Lua scripts
func Load(L *lua.LState) {

	// Register the SQLite function for one-shot queries
	L.SetGlobal("SQLite", L.NewFunction(func(L *lua.LState) int {
		// Check if the optional arguments are given
		query := defaultQuery
		if L.GetTop() >= 1 {
			query = L.ToString(1)
			if query == "" {
				query = defaultQuery
			}
		}
		filename := defaultFilename
		if L.GetTop() >= 2 {
			filename = L.ToString(2)
		}

		db, err := getDB(filename)
		if err != nil {
			logrus.Error("Could not open SQLite database " + filename + ": " + err.Error())
			return 0 // No results
		}

		L.Push(queryRows(L, db, query))
		return 1 // number of results
	}))

	// Register the SQLiteFile class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lSQLiteFileClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, sqlfMethods)

	// The constructor for new SQLiteFile handles takes a filename
	L.SetGlobal("SQLiteFile", L.NewFunction(func(L *lua.LState) int {
		// Check if the optional argument is given
		filename := defaultFilename
		if L.GetTop() >= 1 {
			filename = L.ToString(1)
			if filename == "" {
				filename = defaultFilename
			}
		}

		db, err := getDB(filename)
		if err != nil {
			logrus.Error("Could not open SQLite database " + filename + ": " + err.Error())
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			L.Push(lua.LNumber(1))
			return 3 // Number of returned values
		}

		// Create a new userdata struct
		ud := L.NewUserData()
		ud.Value = &sqliteFileData{db: db, filename: filename}

		L.SetMetatable(ud, L.GetTypeMetatable(lSQLiteFileClass))

		// Return the SQLiteFile object
		L.Push(ud)
		return 1 // Number of returned values
	}))
}
