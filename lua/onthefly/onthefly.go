// Package onthefly provides Lua functions for building HTML, XML and TinySVG documents
package onthefly

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/gopher-lua"
	"github.com/xyproto/onthefly"
	"github.com/xyproto/tinysvg"
)

const (
	// PageClass is an identifier for the Page class in Lua
	PageClass = "Page"

	// TagClass is an identifier for the Tag class in Lua
	TagClass = "Tag"
)

// Get the first argument, "self", and cast it from userdata to a library (which is really a hash map).
func checkTag(L *lua.LState) *onthefly.Tag {
	ud := L.CheckUserData(1)
	if tag, ok := ud.Value.(*onthefly.Tag); ok {
		return tag
	}
	L.ArgError(1, "Tag expected")
	return nil
}

// Create a new onthefly.Tag node. onthefly.Tag data as the first argument is optional.
// Logs an error if the given onthefly.Tag can't be parsed.
// Always returns a onthefly.Tag Node.
func constructTag(L *lua.LState) (*lua.LUserData, error) {
	// Use the first argument as the name of the tag
	name := L.ToString(1)
	if name == "" {
		L.ArgError(1, "tag name expected")
		return nil, errors.New("tag name expected")
	}

	// Create a new onthefly.Tag
	tag := onthefly.NewTag(name)

	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = tag
	L.SetMetatable(ud, L.GetTypeMetatable(TagClass))
	return ud, nil
}

// Take a Tag and a onthefly.Tag path.
// Remove a key from a map. Return true if successful.
func tagAddNewTag(L *lua.LState) int {

	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "tag name expected")
		return 0 // no results
	}

	// Create a new Tag
	newTag := tag.AddNewTag(name)

	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = newTag
	L.SetMetatable(ud, L.GetTypeMetatable(TagClass))

	// Return the newly created userdata
	L.Push(ud)
	return 1 // number of results
}

func tagString(L *lua.LState) int {
	tag := checkTag(L) // arg 1

	// Return the description as a Lua string
	L.Push(lua.LString(tag.String()))
	return 1 // number of results
}

// The hash map methods that are to be registered
var tagMethods = map[string]lua.LGFunction{
	"__tostring": tagString,
	"addNewTag":  tagAddNewTag,
}

// Get the first argument, "self", and cast it from userdata to a library (which is really a hash map).
func checkPage(L *lua.LState) *onthefly.Page {
	ud := L.CheckUserData(1)
	if page, ok := ud.Value.(*onthefly.Page); ok {
		return page
	}
	L.ArgError(1, "Page expected")
	return nil
}

func constructHTML5Page(L *lua.LState) (*lua.LUserData, error) {
	// Use the first argument as the title of the page
	title := L.ToString(1)

	// Create a new HTML5 Page
	page := onthefly.NewHTML5Page(title)

	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = page
	L.SetMetatable(ud, L.GetTypeMetatable(PageClass))

	return ud, nil
}

// Construct an onthefly.Page that represents a TinySVG object
func constructTinySVGPage(L *lua.LState) (*lua.LUserData, error) {
	// Use the first argument as the title of the page
	w := L.ToInt(1)
	h := L.ToInt(2)
	description := L.ToString(3)

	// Create a new TinySVG Page
	document, svgTag := tinysvg.NewTinySVG(w, h)
	svgTag.Describe(description)

	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = document
	L.SetMetatable(ud, L.GetTypeMetatable(PageClass))

	return ud, nil
}

// Create a new onthefly.Page
func constructPage(L *lua.LState) (*lua.LUserData, error) {
	// Use the first argument as the title of the page
	title := L.ToString(1)
	if title == "" {
		L.ArgError(1, "page title expected")
		return nil, errors.New("page title expected")
	}

	// Use the second argument as the name of the root tag
	rootTagName := L.ToString(2)
	if rootTagName == "" {
		L.ArgError(1, "root tag name expected")
		return nil, errors.New("root tag name expected")
	}

	// Create a new onthefly.Page
	page := onthefly.NewPage(title, rootTagName)

	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = page
	L.SetMetatable(ud, L.GetTypeMetatable(PageClass))

	return ud, nil
}

// Return the Page as a string
func pageString(L *lua.LState) int {

	page := checkPage(L) // arg 1

	// Return the Page as a Lua string
	L.Push(lua.LString(page.String()))
	return 1 // number of results
}

// The hash map methods that are to be registered
var pageMethods = map[string]lua.LGFunction{
	"__tostring": pageString,
}

// Load makes functions related to onthefly.Page nodes available to the given Lua state
func Load(L *lua.LState) {

	// Register the Page class and the methods that belongs with it.
	metaTablePage := L.NewTypeMetatable(PageClass)
	metaTablePage.RawSetH(lua.LString("__index"), metaTablePage)
	L.SetFuncs(metaTablePage, pageMethods)

	metaTableTag := L.NewTypeMetatable(TagClass)
	metaTableTag.RawSetH(lua.LString("__index"), metaTableTag)
	L.SetFuncs(metaTableTag, tagMethods)

	// The constructor for Page
	L.SetGlobal("Page", L.NewFunction(func(L *lua.LState) int {
		// Construct a new Page
		userdata, err := constructPage(L)
		if err != nil {
			log.Error(err)
			L.Push(lua.LString(err.Error()))
			return 1 // Number of returned values
		}

		// Return the Lua Page object
		L.Push(userdata)
		return 1 // number of results
	}))

	// The constructor for HTML5 returns a Page
	L.SetGlobal("HTML5", L.NewFunction(func(L *lua.LState) int {
		// Construct a new Page
		userdata, err := constructHTML5Page(L)
		if err != nil {
			log.Error(err)
			L.Push(lua.LString(err.Error()))
			return 1 // Number of returned values
		}

		// Return the Lua Page object
		L.Push(userdata)
		return 1 // number of results
	}))

	// The constructor for TinySVG returns a Page
	L.SetGlobal("TinySVG", L.NewFunction(func(L *lua.LState) int {
		// Construct a new Page
		userdata, err := constructTinySVGPage(L)
		if err != nil {
			log.Error(err)
			L.Push(lua.LString(err.Error()))
			return 1 // Number of returned values
		}

		// Return the Lua Page object
		L.Push(userdata)
		return 1 // number of results
	}))

	// The constructor for Tag
	L.SetGlobal("Tag", L.NewFunction(func(L *lua.LState) int {
		// Construct a new Tag
		userdata, err := constructTag(L)
		if err != nil {
			log.Error(err)
			L.Push(lua.LString(err.Error()))
			return 1 // Number of returned values
		}

		// Return the Lua Tag object
		L.Push(userdata)
		return 1 // number of results
	}))

}
