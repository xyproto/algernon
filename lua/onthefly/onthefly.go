// Package onthefly provides Lua functions for building HTML, XML and TinySVG documents
package onthefly

import (
	"errors"

	"github.com/sirupsen/logrus"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/onthefly"
	"github.com/xyproto/tinysvg"
)

const (
	// PageClass is an identifier for the Page class in Lua
	PageClass = "Page"

	// TagClass is an identifier for the Tag class in Lua
	TagClass = "Tag"
)

// Get the first argument, "self", and cast it from userdata to a Tag.
func checkTag(L *lua.LState) *onthefly.Tag {
	ud := L.CheckUserData(1)
	if tag, ok := ud.Value.(*onthefly.Tag); ok {
		return tag
	}
	L.ArgError(1, "Tag expected")
	return nil
}

// Get the first argument, "self", and cast it from userdata to a Page.
func checkPage(L *lua.LState) *onthefly.Page {
	ud := L.CheckUserData(1)
	if page, ok := ud.Value.(*onthefly.Page); ok {
		return page
	}
	L.ArgError(1, "Page expected")
	return nil
}

// pushSelf re-pushes the first argument (self) so that setters can be chained.
// Always returns 1, to be used as the return value of a Lua function.
func pushSelf(L *lua.LState) int {
	L.Push(L.Get(1))
	return 1
}

// pushTag wraps the given Tag in a Lua userdata with the Tag metatable.
// Pushes nil if the given Tag is nil.
func pushTag(L *lua.LState, tag *onthefly.Tag) {
	if tag == nil {
		L.Push(lua.LNil)
		return
	}
	ud := L.NewUserData()
	ud.Value = tag
	L.SetMetatable(ud, L.GetTypeMetatable(TagClass))
	L.Push(ud)
}

// Create a new onthefly.Tag node.
// Logs an error if the given tag name is empty.
// Always returns a Tag userdata.
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

// Create a new HTML5 onthefly.Page.
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

// Construct a TinySVG document that is wrapped in a Page-compatible userdata.
// NOTE: tinysvg.Document is a separate type from onthefly.Page, so only
// __tostring works on the returned userdata. Use this for SVG generation only.
func constructTinySVGPage(L *lua.LState) (*lua.LUserData, error) {
	// Use the first arguments as the width and height of the SVG
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
		L.ArgError(2, "root tag name expected")
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

// Load makes functions related to onthefly.Page nodes available to the given Lua state
func Load(L *lua.LState) {
	// Register the Page class and the methods that belongs with it.
	metaTablePage := L.NewTypeMetatable(PageClass)
	metaTablePage.RawSetH(lua.LString("__index"), metaTablePage)
	L.SetFuncs(metaTablePage, pageMethods)

	// Register the Tag class and the methods that belongs with it.
	metaTableTag := L.NewTypeMetatable(TagClass)
	metaTableTag.RawSetH(lua.LString("__index"), metaTableTag)
	L.SetFuncs(metaTableTag, tagMethods)

	// The constructor for Page
	L.SetGlobal("Page", L.NewFunction(func(L *lua.LState) int {
		// Construct a new Page
		userdata, err := constructPage(L)
		if err != nil {
			logrus.Error(err)
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
			logrus.Error(err)
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
			logrus.Error(err)
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
			logrus.Error(err)
			L.Push(lua.LString(err.Error()))
			return 1 // Number of returned values
		}

		// Return the Lua Tag object
		L.Push(userdata)
		return 1 // number of results
	}))

}
