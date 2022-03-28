package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"

	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/pongo2"
	"github.com/yuin/gluamapper"
	"github.com/yuin/gopher-lua"

	log "github.com/sirupsen/logrus"
)

// LoadServeFile exposes functions for serving other files to Lua
func (ac *Config) LoadServeFile(w http.ResponseWriter, req *http.Request, L *lua.LState, filename string) {

	// Serve a file in the scriptdir
	L.SetGlobal("serve", L.NewFunction(func(L *lua.LState) int {
		scriptdir := filepath.Dir(filename)
		serveFilename := filepath.Join(scriptdir, L.ToString(1))
		dataFilename := filepath.Join(scriptdir, ac.defaultLuaDataFilename)
		if L.GetTop() >= 2 {
			// Optional argument for using a different file than "data.lua"
			dataFilename = filepath.Join(scriptdir, L.ToString(2))
		}
		if !ac.fs.Exists(serveFilename) {
			log.Error("Could not serve " + serveFilename + ". File not found.")
			return 0 // Number of results
		}
		if ac.fs.IsDir(serveFilename) {
			log.Error("Could not serve " + serveFilename + ". Not a file.")
			return 0 // Number of results
		}
		ac.FilePage(w, req, serveFilename, dataFilename)
		return 0 // Number of results
	}))

	// Output text as rendered Pongo2, using a po2 file and an optional table
	L.SetGlobal("serve2", L.NewFunction(func(L *lua.LState) int {
		scriptdir := filepath.Dir(filename)

		// Use the first argument as the template and the second argument as the data map
		templateFilename := filepath.Join(scriptdir, L.CheckString(1))
		ext := filepath.Ext(strings.ToLower(templateFilename))

		templateData, err := ac.cache.Read(templateFilename, ac.shouldCache(ext))
		if err != nil {
			if ac.debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", templateFilename, err)
			} else {
				log.Errorf("Unable to read %s: %s", templateFilename, err)
			}
			return 0 // number of restuls
		}
		templateString := templateData.String()

		// If a table is given as the second argument, fill pongoMap with keys and values
		pongoMap := make(pongo2.Context)

		if L.GetTop() == 2 {
			luaTable := L.CheckTable(2)
			goMap := gluamapper.ToGoValue(luaTable, gluamapper.Option{
				NameFunc: func(s string) string {
					return s
				},
			}).(map[string]interface{})
			pongoMap = pongo2.Context(goMap)
			//fmt.Println("PONGOMAP", pongoMap, "LUA TABLE", luaTable)
		} else if L.GetTop() > 2 {
			log.Error("Too many arguments given to the serve2 function")
			return 0 // number of restuls
		}

		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Use the buffer as a template.
		// Options are "Pretty printing, but without line numbers."
		tpl, err := pongo2.FromString(templateString)
		if err != nil {
			if ac.debugMode {
				fmt.Fprint(w, "Could not compile Pongo2 template:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile Pongo2 template:\n%s\n%s", err, buf.String())
			}
			return 0 // number of results
		}
		// nil is the template context (variables etc in a map)
		if err := tpl.ExecuteWriter(pongoMap, w); err != nil {
			if ac.debugMode {
				fmt.Fprint(w, "Could not compile Pongo2:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile Pongo2:\n%s\n%s", err, buf.String())
			}
		}
		return 0 // number of results
	}))

	// Get the rendered contents of a file in the scriptdir. Discards HTTP headers.
	L.SetGlobal("render", L.NewFunction(func(L *lua.LState) int {
		scriptdir := filepath.Dir(filename)
		serveFilename := filepath.Join(scriptdir, L.ToString(1))
		dataFilename := filepath.Join(scriptdir, ac.defaultLuaDataFilename)
		if L.GetTop() >= 2 {
			// Optional argument for using a different file than "data.lua"
			dataFilename = filepath.Join(scriptdir, L.ToString(2))
		}
		if !ac.fs.Exists(serveFilename) {
			log.Error("Could not render " + serveFilename + ". File not found.")
			return 0 // Number of results
		}
		if ac.fs.IsDir(serveFilename) {
			log.Error("Could not render " + serveFilename + ". Not a file.")
			return 0 // Number of results
		}

		// Render the filename to a httptest.Recorder
		recorder := httptest.NewRecorder()
		ac.FilePage(recorder, req, serveFilename, dataFilename)

		// Return the recorder as a string
		L.Push(lua.LString(utils.RecorderToString(recorder)))
		return 1 // Number of results
	}))

}
