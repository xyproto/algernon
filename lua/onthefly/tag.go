package onthefly

import (
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/onthefly"
)

// Return the Tag as a string
func tagString(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	L.Push(lua.LString(tag.String()))
	return 1 // number of results
}

// Take a Tag and a tag name.
// Adds a new child tag to the Tag.
// Returns the newly added Tag.
func tagAddNewTag(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "tag name expected")
		return 0 // no results
	}
	pushTag(L, tag.AddNewTag(name))
	return 1 // number of results
}

// Take a Tag and a Tag.
// Adds the second Tag as a child of the first. Returns the parent Tag.
func tagAddTag(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	ud := L.CheckUserData(2)
	child, ok := ud.Value.(*onthefly.Tag)
	if !ok {
		L.ArgError(2, "Tag expected")
		return 0 // no results
	}
	tag.AddTag(child)
	return pushSelf(L)
}

// Take a Tag, a CSS style name and a CSS style value. Returns the Tag.
func tagAddStyle(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "style name expected")
		return 0 // no results
	}
	value := L.ToString(3)
	tag.AddStyle(name, value)
	return pushSelf(L)
}

// Take a Tag, an attribute name and an attribute value. Returns the Tag.
func tagAddAttrib(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "attribute name expected")
		return 0 // no results
	}
	value := L.ToString(3)
	tag.AddAttrib(name, value)
	return pushSelf(L)
}

// Take a Tag and an attribute name. Adds an attribute without a value.
// Returns the Tag.
func tagAddSingularAttrib(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "attribute name expected")
		return 0 // no results
	}
	tag.AddSingularAttrib(name)
	return pushSelf(L)
}

// Return the CSS for this Tag as a string.
func tagGetCSS(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	L.Push(lua.LString(tag.GetCSS()))
	return 1 // number of results
}

// Return the attribute string for this Tag.
func tagGetAttrString(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	L.Push(lua.LString(tag.GetAttrString()))
	return 1 // number of results
}

// Take a Tag and content to add between the opening and closing tag.
// Returns the Tag.
func tagAddContent(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	tag.AddContent(L.ToString(2))
	return pushSelf(L)
}

// Take a Tag and content to append after any existing content / child tags.
// Returns the Tag.
func tagAppendContent(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	tag.AppendContent(L.ToString(2))
	return pushSelf(L)
}

// Return the number of children of this Tag.
func tagCountChildren(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	L.Push(lua.LNumber(tag.CountChildren()))
	return 1 // number of results
}

// Return the number of siblings of this Tag.
func tagCountSiblings(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	L.Push(lua.LNumber(tag.CountSiblings()))
	return 1 // number of results
}

// Return the last child of this Tag.
func tagLastChild(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	pushTag(L, tag.LastChild())
	return 1 // number of results
}

// Return the first child of this Tag.
func tagFirstChild(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	pushTag(L, tag.GetFirstChild())
	return 1 // number of results
}

// Return the next sibling of this Tag.
func tagNextSibling(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	pushTag(L, tag.GetNextSibling())
	return 1 // number of results
}

// Return all children of this Tag as a Lua table of Tag userdata.
func tagGetChildren(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	children := tag.GetChildren()
	table := L.CreateTable(len(children), 0)
	for _, child := range children {
		ud := L.NewUserData()
		ud.Value = child
		L.SetMetatable(ud, L.GetTypeMetatable(TagClass))
		table.Append(ud)
	}
	L.Push(table)
	return 1 // number of results
}

// Return the name of this Tag.
func tagGetName(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	L.Push(lua.LString(tag.GetName()))
	return 1 // number of results
}

// Return the content of this Tag.
func tagGetContent(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	L.Push(lua.LString(tag.GetContent()))
	return 1 // number of results
}

// Replace the content of this Tag with the given string. Returns the Tag.
func tagSetContent(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	tag.SetContent(L.ToString(2))
	return pushSelf(L)
}

// Take a Tag and a name. Returns the first matching descendant Tag or nil.
func tagGetTag(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "tag name expected")
		return 0 // no results
	}
	found, err := tag.GetTag(name)
	if err != nil {
		L.Push(lua.LNil)
		return 1 // number of results
	}
	pushTag(L, found)
	return 1 // number of results
}

// Take a Tag and an attribute name. Returns true if the attribute exists.
func tagHasAttribute(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "attribute name expected")
		return 0 // no results
	}
	L.Push(lua.LBool(tag.HasAttribute(name)))
	return 1 // number of results
}

// Take a Tag and an attribute name.
// Returns the attribute value and a boolean indicating if it existed.
func tagGetAttribute(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "attribute name expected")
		return 0 // no results
	}
	value, ok := tag.GetAttribute(name)
	L.Push(lua.LString(value))
	L.Push(lua.LBool(ok))
	return 2 // number of results
}

// Take a Tag and an attribute name. Removes that attribute. Returns the Tag.
func tagRemoveAttribute(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "attribute name expected")
		return 0 // no results
	}
	tag.RemoveAttribute(name)
	return pushSelf(L)
}

// Remove all children of this Tag. Returns the Tag.
func tagClearChildren(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	tag.ClearChildren()
	return pushSelf(L)
}

// Return a deep copy of this Tag.
func tagClone(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	pushTag(L, tag.CloneTag())
	return 1 // number of results
}

// Take a Tag and a name. Returns the first descendant child Tag with the given name, or nil.
func tagFindChildByName(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "tag name expected")
		return 0 // no results
	}
	pushTag(L, tag.FindChildByName(name))
	return 1 // number of results
}

// Take a Tag, an attribute name and an attribute value.
// Returns the first descendant child Tag matching the attribute, or nil.
func tagFindChildByAttribute(L *lua.LState) int {
	tag := checkTag(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "attribute name expected")
		return 0 // no results
	}
	value := L.ToString(3)
	pushTag(L, tag.FindChildByAttribute(name, value))
	return 1 // number of results
}

// The Tag method table that is to be registered
var tagMethods = map[string]lua.LGFunction{
	"__tostring":           tagString,
	"addNewTag":            tagAddNewTag,
	"addTag":               tagAddTag,
	"addStyle":             tagAddStyle,
	"addAttrib":            tagAddAttrib,
	"addSingularAttrib":    tagAddSingularAttrib,
	"getCSS":               tagGetCSS,
	"getAttrString":        tagGetAttrString,
	"addContent":           tagAddContent,
	"appendContent":        tagAppendContent,
	"countChildren":        tagCountChildren,
	"countSiblings":        tagCountSiblings,
	"lastChild":            tagLastChild,
	"firstChild":           tagFirstChild,
	"nextSibling":          tagNextSibling,
	"children":             tagGetChildren,
	"name":                 tagGetName,
	"content":              tagGetContent,
	"setContent":           tagSetContent,
	"tag":                  tagGetTag,
	"hasAttribute":         tagHasAttribute,
	"getAttribute":         tagGetAttribute,
	"removeAttribute":      tagRemoveAttribute,
	"clearChildren":        tagClearChildren,
	"clone":                tagClone,
	"findChildByName":      tagFindChildByName,
	"findChildByAttribute": tagFindChildByAttribute,
}
