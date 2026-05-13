package sqlite

import (
	"os"
	"testing"

	lua "github.com/xyproto/gopher-lua"
)

// newState creates a Lua state with the SQLite functions loaded
func newState(t *testing.T) *lua.LState {
	t.Helper()
	L := lua.NewState()
	Load(L)
	return L
}

// tempDB returns a temporary database filename that is cleaned up after the test
func tempDB(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestSQLiteVersion(t *testing.T) {
	L := newState(t)
	defer L.Close()

	if err := L.DoString(`result = SQLite()`); err != nil {
		t.Fatal(err)
	}

	result := L.GetGlobal("result")
	tbl, ok := result.(*lua.LTable)
	if !ok {
		t.Fatal("expected table result from SQLite()")
	}

	// The default query is SELECT sqlite_version(), which returns one row with one column
	row := tbl.RawGetInt(1)
	if row == lua.LNil {
		t.Fatal("expected at least one row")
	}
}

func TestSQLiteCustomQuery(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`result = SQLite("SELECT 1+1 AS answer", "` + dbFile + `")`); err != nil {
		t.Fatal(err)
	}

	result := L.GetGlobal("result")
	tbl, ok := result.(*lua.LTable)
	if !ok {
		t.Fatal("expected table result")
	}
	row := tbl.RawGetInt(1)
	rowTbl, ok := row.(*lua.LTable)
	if !ok {
		t.Fatal("expected table row")
	}
	answer := rowTbl.RawGetString("answer")
	if answer.String() != "2" {
		t.Errorf("expected answer=2, got %q", answer.String())
	}
}

func TestSQLiteFileOpenClose(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		ok = db:close()
	`); err != nil {
		t.Fatal(err)
	}

	ok := L.GetGlobal("ok")
	if ok != lua.LTrue {
		t.Error("expected close to return true")
	}
}

func TestSQLiteFileToString(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		name = tostring(db)
	`); err != nil {
		t.Fatal(err)
	}

	name := L.GetGlobal("name")
	if name.String() != dbFile {
		t.Errorf("expected %q, got %q", dbFile, name.String())
	}
}

func TestSQLiteFileExecAndQuery(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		db:exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
		db:exec("INSERT INTO test (name) VALUES (?)", {"Alice"})
		db:exec("INSERT INTO test (name) VALUES (?)", {"Bob"})
		rows = db:query("SELECT name FROM test ORDER BY name")
	`); err != nil {
		t.Fatal(err)
	}

	rows := L.GetGlobal("rows")
	tbl, ok := rows.(*lua.LTable)
	if !ok {
		t.Fatal("expected table result")
	}
	if tbl.Len() != 2 {
		t.Fatalf("expected 2 rows, got %d", tbl.Len())
	}

	first := tbl.RawGetInt(1).(*lua.LTable).RawGetString("name")
	if first.String() != "Alice" {
		t.Errorf("expected Alice, got %q", first.String())
	}
}

func TestSQLiteFileExecReturnValues(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		db:exec("CREATE TABLE t (v TEXT)")
		n, ok = db:exec("INSERT INTO t VALUES ('a')")
	`); err != nil {
		t.Fatal(err)
	}

	n := L.GetGlobal("n")
	ok := L.GetGlobal("ok")
	if n.String() != "1" {
		t.Errorf("expected affected=1, got %q", n.String())
	}
	if ok != lua.LTrue {
		t.Error("expected ok=true")
	}
}

func TestSQLiteFileQueryParams(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		db:exec("CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT)")
		db:exec("INSERT INTO items (val) VALUES (?)", {"hello"})
		db:exec("INSERT INTO items (val) VALUES (?)", {"world"})
		rows = db:query("SELECT val FROM items WHERE val = ?", {"hello"})
	`); err != nil {
		t.Fatal(err)
	}

	rows := L.GetGlobal("rows")
	tbl := rows.(*lua.LTable)
	if tbl.Len() != 1 {
		t.Fatalf("expected 1 row, got %d", tbl.Len())
	}
}

func TestSQLiteFileTransaction(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		db:exec("CREATE TABLE t (v TEXT)")
		ok = db:transaction(function()
			db:exec("INSERT INTO t VALUES ('in_tx')")
		end)
		rows = db:query("SELECT v FROM t")
	`); err != nil {
		t.Fatal(err)
	}

	ok := L.GetGlobal("ok")
	if ok != lua.LTrue {
		t.Error("expected transaction to return true")
	}
	rows := L.GetGlobal("rows")
	tbl := rows.(*lua.LTable)
	if tbl.Len() != 1 {
		t.Fatalf("expected 1 row after transaction, got %d", tbl.Len())
	}
	val := tbl.RawGetInt(1).(*lua.LTable).RawGetString("v")
	if val.String() != "in_tx" {
		t.Errorf("expected 'in_tx', got %q", val.String())
	}
}

func TestSQLiteFileTransactionRollback(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		db:exec("CREATE TABLE t (v TEXT)")
		ok = db:transaction(function()
			db:exec("INSERT INTO t VALUES ('should_rollback')")
			error("force rollback")
		end)
		rows = db:query("SELECT v FROM t")
	`); err != nil {
		t.Fatal(err)
	}

	ok := L.GetGlobal("ok")
	if ok != lua.LFalse {
		t.Error("expected transaction to return false on error")
	}
	rows := L.GetGlobal("rows")
	tbl := rows.(*lua.LTable)
	if tbl.Len() != 0 {
		t.Errorf("expected 0 rows after rollback, got %d", tbl.Len())
	}
}

func TestSQLiteFileDocAdd(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		id = db:add("users", {name = "Alice", age = 30})
	`); err != nil {
		t.Fatal(err)
	}

	id := L.GetGlobal("id")
	if id.String() != "1" {
		t.Errorf("expected id=1, got %q", id.String())
	}
}

func TestSQLiteFileDocGet(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		id = db:add("users", {name = "Alice", age = 30})
		doc = db:get("users", id)
		empty = db:get("users", "999")
	`); err != nil {
		t.Fatal(err)
	}

	doc := L.GetGlobal("doc").(*lua.LTable)
	name := doc.RawGetString("name")
	if name.String() != "Alice" {
		t.Errorf("expected Alice, got %q", name.String())
	}

	empty := L.GetGlobal("empty").(*lua.LTable)
	if empty.Len() != 0 {
		t.Errorf("expected empty table for missing ID, got %d fields", empty.Len())
	}
}

func TestSQLiteFileDocDocs(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		db:add("users", {name = "Alice", age = 30})
		db:add("users", {name = "Bob", age = 25})
		all = db:docs("users")
		filtered = db:docs("users", {name = "Bob"})
	`); err != nil {
		t.Fatal(err)
	}

	all := L.GetGlobal("all").(*lua.LTable)
	if all.Len() != 2 {
		t.Fatalf("expected 2 docs, got %d", all.Len())
	}

	filtered := L.GetGlobal("filtered").(*lua.LTable)
	if filtered.Len() != 1 {
		t.Fatalf("expected 1 filtered doc, got %d", filtered.Len())
	}
	doc := filtered.RawGetInt(1).(*lua.LTable)
	name := doc.RawGetString("name")
	if name.String() != "Bob" {
		t.Errorf("expected Bob, got %q", name.String())
	}
}

func TestSQLiteFileDocDel(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		db:add("users", {name = "Alice"})
		db:add("users", {name = "Bob"})
		ok = db:del("users", {name = "Alice"})
		remaining = db:docs("users")
	`); err != nil {
		t.Fatal(err)
	}

	ok := L.GetGlobal("ok")
	if ok != lua.LTrue {
		t.Error("expected del to return true")
	}
	remaining := L.GetGlobal("remaining").(*lua.LTable)
	if remaining.Len() != 1 {
		t.Fatalf("expected 1 remaining doc, got %d", remaining.Len())
	}
}

func TestSQLiteFileDocUpdate(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		db:add("users", {name = "Alice", age = 30})
		ok = db:update("users", {name = "Alice"}, {age = 31})
		docs = db:docs("users", {name = "Alice"})
	`); err != nil {
		t.Fatal(err)
	}

	ok := L.GetGlobal("ok")
	if ok != lua.LTrue {
		t.Error("expected update to return true")
	}
	docs := L.GetGlobal("docs").(*lua.LTable)
	doc := docs.RawGetInt(1).(*lua.LTable)
	age := doc.RawGetString("age")
	if age.String() != "31" {
		t.Errorf("expected age=31, got %q", age.String())
	}
}

func TestSQLiteFileDocLen(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		db:add("items", {x = 1})
		db:add("items", {x = 2})
		db:add("items", {x = 3})
		n = db:len("items")
	`); err != nil {
		t.Fatal(err)
	}

	n := L.GetGlobal("n")
	if n.String() != "3" {
		t.Errorf("expected len=3, got %q", n.String())
	}
}

func TestSQLiteFileInvalidCollectionName(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	// SQL injection attempt in collection name should fail safely
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		ok = db:del("users; DROP TABLE users--", {name = "x"})
		n = db:len("users; DROP TABLE users--")
	`); err != nil {
		t.Fatal(err)
	}

	ok := L.GetGlobal("ok")
	if ok != lua.LFalse {
		t.Error("expected del with invalid collection to return false")
	}
	n := L.GetGlobal("n")
	if n.String() != "0" {
		t.Error("expected len with invalid collection to return 0")
	}
}

func TestSQLiteFileConnectionReuse(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	// Opening the same file twice should reuse the connection
	if err := L.DoString(`
		db1 = SQLiteFile("` + dbFile + `")
		db1:exec("CREATE TABLE test (v TEXT)")
		db1:exec("INSERT INTO test VALUES ('hello')")
		db2 = SQLiteFile("` + dbFile + `")
		rows = db2:query("SELECT v FROM test")
	`); err != nil {
		t.Fatal(err)
	}

	rows := L.GetGlobal("rows").(*lua.LTable)
	if rows.Len() != 1 {
		t.Errorf("expected 1 row via reused connection, got %d", rows.Len())
	}
}

func TestSQLiteFileDocStoreInTransaction(t *testing.T) {
	L := newState(t)
	defer L.Close()

	dbFile := tempDB(t)
	if err := L.DoString(`
		db = SQLiteFile("` + dbFile + `")
		ok = db:transaction(function()
			db:add("items", {name = "a"})
			db:add("items", {name = "b"})
		end)
		n = db:len("items")
	`); err != nil {
		t.Fatal(err)
	}

	ok := L.GetGlobal("ok")
	if ok != lua.LTrue {
		t.Error("expected transaction to return true")
	}
	n := L.GetGlobal("n")
	if n.String() != "2" {
		t.Errorf("expected 2 items after transaction, got %q", n.String())
	}
}
