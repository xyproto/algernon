package sqlite3

import (
	"bytes"
	"encoding/base64"
	"sync"

	"github.com/ncruces/go-sqlite3/internal/errutil"
)

var (
	// +checklocks:extRegistryMtx
	extRegistry    []func(*Conn) error
	extRegistryMtx sync.RWMutex
)

// AutoExtension causes the entryPoint function to be invoked
// for each new database connection that is created.
//
// https://sqlite.org/c3ref/auto_extension.html
func AutoExtension(entryPoint func(*Conn) error) {
	extRegistryMtx.Lock()
	extRegistry = append(extRegistry, entryPoint)
	extRegistryMtx.Unlock()
}

func initExtensions(c *Conn) error {
	c.base64()
	extRegistryMtx.RLock()
	defer extRegistryMtx.RUnlock()
	for _, f := range extRegistry {
		if err := f(c); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) base64() error {
	return c.CreateFunction("base64", 1, DETERMINISTIC, func(ctx Context, arg ...Value) {
		switch a := arg[0]; a.Type() {
		case NULL:

		case BLOB:
			data := a.RawBlob()
			code := base64.StdEncoding
			size := int64(code.EncodedLen(len(data)))
			if size > _MAX_LENGTH {
				ctx.ResultError(TOOBIG)
				return
			}
			ptr := c.wrp.New(size)
			if size > 0 {
				code.Encode(c.wrp.Bytes(ptr, size), data)
			}
			ctx.c.wrp.Xsqlite3_result_text_go(int32(ctx.handle), int32(ptr), size)

		case TEXT:
			data := a.RawText()
			data = bytes.Trim(data, " \t\n\v\f\r")
			data = bytes.TrimRight(data, "=")
			code := base64.RawStdEncoding
			size := int64(code.DecodedLen(len(data)))
			if size > _MAX_LENGTH {
				ctx.ResultError(TOOBIG)
				return
			}
			ptr := c.wrp.New(size)
			if size > 0 {
				n, _ := code.Decode(c.wrp.Bytes(ptr, size), data)
				size = int64(n)
			}
			ctx.c.wrp.Xsqlite3_result_blob_go(int32(ctx.handle), int32(ptr), size)

		default:
			ctx.ResultError(errutil.ErrorString("base64: accepts only blob or text"))
		}
	})
}

// ExtensionLibrary represents a dynamically linked SQLite extension.
type ExtensionLibrary interface {
	Xsqlite3_extension_init(db, _, _ int32) int32
}

// ExtensionInfo returns values needed to load a dynamically linked SQLite extension.
type ExtensionInfo func() (memorySize, memoryAlignment, tableSize, tableAlignment int64)

type extEnv struct {
	*env
	memoryBase int32
	tableBase  int32
}

func (e *extEnv) X__memory_base() *int32 { return &e.memoryBase }
func (e *extEnv) X__table_base() *int32  { return &e.tableBase }

// ExtensionInit loads an SQLite extension library.
//
// https://sqlite.org/loadext.html
func ExtensionInit[Env any, Mod ExtensionLibrary](db *Conn, init func(env Env) Mod, info ExtensionInfo) error {
	memSize, memAlign, tableSize, tableAlign := info()

	var memBase int32
	if memSize > 0 {
		memBase = db.wrp.Xaligned_alloc(int32(memAlign), int32(memSize))
		if memBase == 0 {
			panic(errutil.OOMErr)
		}
	}

	var tableBase int
	if tableSize > 0 {
		// Round up to the alignment.
		rnd := int(tableAlign) - 1
		tab := db.wrp.X__indirect_function_table()
		tableBase = (len(*tab) + rnd) &^ rnd
		if add := tableBase + int(tableSize) - len(*tab); add > 0 {
			*tab = append(*tab, make([]any, add)...)
		}
	}

	e := &extEnv{
		env:        &env{db.wrp},
		memoryBase: memBase,
		tableBase:  int32(tableBase),
	}

	mod := init(any(e).(Env))
	if opt, ok := any(mod).(interface{ X__wasm_apply_data_relocs() }); ok {
		opt.X__wasm_apply_data_relocs()
	}
	if opt, ok := any(mod).(interface{ X__wasm_call_ctors() }); ok {
		opt.X__wasm_call_ctors()
	}
	rc := mod.Xsqlite3_extension_init(int32(db.handle), 0, 0)
	return db.error(res_t(rc))
}
