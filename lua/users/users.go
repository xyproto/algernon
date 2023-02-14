// Package users provides Lua functions for dealing with users and permissions
package users

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/gopher-lua"
	"github.com/xyproto/pinterface"
)

// Load makes functions related to users and permissions available to Lua scripts
func Load(w http.ResponseWriter, req *http.Request, L *lua.LState, userstate pinterface.IUserState) {
	// Check if the current user has "user rights", returns bool
	// Takes no arguments
	L.SetGlobal("UserRights", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LBool(userstate.UserRights(req)))
		return 1 // number of results
	}))
	// Check if the given username exists, returns bool
	// Takes a username
	L.SetGlobal("HasUser", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		L.Push(lua.LBool(userstate.HasUser(username)))
		return 1 // number of results
	}))
	// Check if the given unconfirmed username exists, returns bool
	// Takes a username
	L.SetGlobal("HasUnconfirmedUser", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		found := false
		unconfirmedUsernames, err := userstate.AllUnconfirmedUsernames()
		if err != nil {
			log.Warn("Could not read the lsit of unconfirmed usernames.")
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		for _, unconfirmedUsername := range unconfirmedUsernames {
			if unconfirmedUsername == username {
				found = true
				break
			}
		}
		L.Push(lua.LBool(found))
		return 1 // number of results
	}))

	// Get the value from the given boolean field, returns bool
	// Takes a username and field name
	L.SetGlobal("BooleanField", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		fieldname := L.ToString(2)
		L.Push(lua.LBool(userstate.BooleanField(username, fieldname)))
		return 1 // number of results
	}))
	// Save a value as a boolean field, returns nothing
	// Takes a username, field name and boolean value
	L.SetGlobal("SetBooleanField", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		fieldname := L.ToString(2)
		value := L.ToBool(3)
		userstate.SetBooleanField(username, fieldname, value)
		return 0 // number of results
	}))
	// Check if a given username is confirmed, returns a bool
	// Takes a username
	L.SetGlobal("IsConfirmed", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		L.Push(lua.LBool(userstate.IsConfirmed(username)))
		return 1 // number of results
	}))
	// Check if a given username is logged in, returns a bool
	// Takes a username
	L.SetGlobal("IsLoggedIn", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		L.Push(lua.LBool(userstate.IsLoggedIn(username)))
		return 1 // number of results
	}))
	// Check if the current user has "admin rights", returns a bool
	// Takes no arguments.
	L.SetGlobal("AdminRights", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LBool(userstate.AdminRights(req)))
		return 1 // number of results
	}))
	// Check if a given username is an admin, returns a bool
	// Takes a username
	L.SetGlobal("IsAdmin", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		L.Push(lua.LBool(userstate.IsAdmin(username)))
		return 1 // number of results
	}))
	// Get the username stored in a cookie, or an empty string
	// Takes no arguments
	L.SetGlobal("UsernameCookie", L.NewFunction(func(L *lua.LState) int {
		username, err := userstate.UsernameCookie(req)
		var result lua.LString
		if err != nil {
			result = lua.LString("")
		} else {
			result = lua.LString(username)
		}
		L.Push(result)
		return 1 // number of results
	}))
	// Store the username in a cookie, returns true if successful
	// Takes a username
	L.SetGlobal("SetUsernameCookie", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		L.Push(lua.LBool(nil == userstate.SetUsernameCookie(w, username)))
		return 1 // number of results
	}))
	// Clear the user cookie. The result depends on the browser.
	L.SetGlobal("ClearCookie", L.NewFunction(func(L *lua.LState) int {
		userstate.ClearCookie(w)
		return 0 // number of results
	}))
	// Get the username stored in a cookie, or an empty string
	// Takes no arguments
	L.SetGlobal("AllUsernames", L.NewFunction(func(L *lua.LState) int {
		usernames, err := userstate.AllUsernames()
		var table *lua.LTable
		if err != nil {
			table = L.NewTable()
		} else {
			table = convert.Strings2table(L, usernames)
		}
		L.Push(table)
		return 1 // number of results
	}))
	// Get the email for a given username, or an empty string
	// Takes a username
	L.SetGlobal("Email", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		email, err := userstate.Email(username)
		var result lua.LString
		if err != nil {
			result = lua.LString("")
		} else {
			result = lua.LString(email)
		}
		L.Push(result)
		return 1 // number of results
	}))
	// Get the password hash for a given username, or an empty string
	// Takes a username
	L.SetGlobal("PasswordHash", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		pw, err := userstate.PasswordHash(username)
		var result lua.LString
		if err != nil {
			result = lua.LString("")
		} else {
			result = lua.LString(pw)
		}
		L.Push(result)
		return 1 // number of results
	}))
	// Get all unconfirmed usernames
	// Takes no arguments
	L.SetGlobal("AllUnconfirmedUsernames", L.NewFunction(func(L *lua.LState) int {
		usernames, err := userstate.AllUnconfirmedUsernames()
		var table *lua.LTable
		if err != nil {
			table = L.NewTable()
		} else {
			table = convert.Strings2table(L, usernames)
		}
		L.Push(table)
		return 1 // number of results
	}))
	// Get the existing confirmation code for a given user, or an empty string.
	L.SetGlobal("ConfirmationCode", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		pw, err := userstate.ConfirmationCode(username)
		if err != nil {
			log.Error(err)
			L.Push(lua.LString(""))
			return 1 // number of results
		}
		L.Push(lua.LString(pw))
		return 1 // number of results
	}))
	// Add a user to the list of unconfirmed users, returns nothing
	// Takes a username and a confirmation code
	L.SetGlobal("AddUnconfirmed", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		confirmationCode := L.ToString(2)
		userstate.AddUnconfirmed(username, confirmationCode)
		return 0 // number of results
	}))
	// Remove a user from the list of unconfirmed users, returns nothing
	// Takes a username
	L.SetGlobal("RemoveUnconfirmed", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		userstate.RemoveUnconfirmed(username)
		return 0 // number of results
	}))
	// Mark a user as confirmed, returns nothing
	// Takes a username
	L.SetGlobal("MarkConfirmed", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		userstate.MarkConfirmed(username)
		return 0 // number of results
	}))
	// Removes a user, returns nothing
	// Takes a username
	L.SetGlobal("RemoveUser", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		userstate.RemoveUser(username)
		return 0 // number of results
	}))
	// Make a user an admin, returns nothing
	// Takes a username
	L.SetGlobal("SetAdminStatus", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		userstate.SetAdminStatus(username)
		return 0 // number of results
	}))
	// Make an admin user a regular user, returns nothing
	// Takes a username
	L.SetGlobal("RemoveAdminStatus", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		userstate.RemoveAdminStatus(username)
		return 0 // number of results
	}))
	// Add a user, returns nothing
	// Takes a username, password and email
	L.SetGlobal("AddUser", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		password := L.ToString(2)
		email := L.ToString(3)
		userstate.AddUser(username, password, email)
		return 0 // number of results
	}))
	// Set a user as logged in on the server (not cookie), returns nothing
	// Takes a username
	L.SetGlobal("SetLoggedIn", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		userstate.SetLoggedIn(username)
		return 0 // number of results
	}))
	// Set a user as logged out on the server (not cookie), returns nothing
	// Takes a username
	L.SetGlobal("SetLoggedOut", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		userstate.SetLoggedOut(username)
		return 0 // number of results
	}))
	// Log in a user, both on the server and with a cookie.
	// Returns true of successful.
	// Takes a username
	L.SetGlobal("Login", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		L.Push(lua.LBool(nil == userstate.Login(w, username)))
		return 1 // number of results
	}))
	// Logs out a user, on the server (which is enough). Returns nothing
	// Takes a username
	L.SetGlobal("Logout", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		userstate.Logout(username)
		return 0 // number of results
	}))
	// Get the current username, from the cookie
	// Takes nothing
	L.SetGlobal("Username", L.NewFunction(func(L *lua.LState) int {
		username := userstate.Username(req)
		L.Push(lua.LString(username))
		return 1 // number of results
	}))

	// Get the current cookie timeout
	// Takes a username
	L.SetGlobal("CookieTimeout", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		L.Push(lua.LNumber(userstate.CookieTimeout(username)))
		return 1 // number of results
	}))
	// Set the current cookie timeout
	// Takes a timeout number, measured in seconds
	L.SetGlobal("SetCookieTimeout", L.NewFunction(func(L *lua.LState) int {
		timeout := int64(L.ToNumber(1))
		userstate.SetCookieTimeout(timeout)
		return 0 // number of results
	}))
	// Get the current cookie secret
	// Takes nothing, returns a string
	L.SetGlobal("CookieSecret", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(userstate.CookieSecret()))
		return 1 // number of results
	}))
	// Set the current cookie secret
	// Takes a string, returns nothing
	L.SetGlobal("SetCookieSecret", L.NewFunction(func(L *lua.LState) int {
		secret := L.ToString(1)
		userstate.SetCookieSecret(secret)
		return 0 // number of results
	}))
	// Get the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
	// Takes nothing
	L.SetGlobal("PasswordAlgo", L.NewFunction(func(L *lua.LState) int {
		algorithm := userstate.PasswordAlgo()
		L.Push(lua.LString(algorithm))
		return 1 // number of results
	}))
	// Set the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
	// Takes a string
	L.SetGlobal("SetPasswordAlgo", L.NewFunction(func(L *lua.LState) int {
		algorithm := L.ToString(1)
		userstate.SetPasswordAlgo(algorithm)
		return 0 // number of results
	}))
	// Change the password for a user
	L.SetGlobal("SetPassword", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		password := L.ToString(2)
		userstate.SetPassword(username, password)
		return 0 // number of results
	}))

	// Hash the password, returns a string
	// Takes a username and password (username can be used for salting)
	L.SetGlobal("HashPassword", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		password := L.ToString(2)
		L.Push(lua.LString(userstate.HashPassword(username, password)))
		return 1 // number of results
	}))
	// Check if a given username and password is correct, returns a bool
	// Takes a username and password
	L.SetGlobal("CorrectPassword", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		password := L.ToString(2)
		L.Push(lua.LBool(userstate.CorrectPassword(username, password)))
		return 1 // number of results
	}))
	// Checks if a confirmation code is already in use, returns a bool
	// Takes a confirmation code
	L.SetGlobal("AlreadyHasConfirmationCode", L.NewFunction(func(L *lua.LState) int {
		confirmationCode := L.ToString(1)
		L.Push(lua.LBool(userstate.AlreadyHasConfirmationCode(confirmationCode)))
		return 1 // number of results
	}))
	// Find a username based on a given confirmation code, or returns an empty string
	// Takes a confirmation code
	L.SetGlobal("FindUserByConfirmationCode", L.NewFunction(func(L *lua.LState) int {
		confirmationCode := L.ToString(1)
		username, err := userstate.FindUserByConfirmationCode(confirmationCode)
		var result lua.LString
		if err != nil {
			log.Warn(err)
			result = lua.LString("")
		} else {
			result = lua.LString(username)
		}
		L.Push(result)
		return 1 // number of results
	}))
	// Mark a user as confirmed, returns nothing
	// Takes a username
	L.SetGlobal("Confirm", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		userstate.Confirm(username)
		return 0 // number of results
	}))
	// Mark a user as confirmed, returns true if successful.
	// Takes a confirmation code.
	L.SetGlobal("ConfirmUserByConfirmationCode", L.NewFunction(func(L *lua.LState) int {
		confirmationCode := L.ToString(1)
		L.Push(lua.LBool(nil == userstate.ConfirmUserByConfirmationCode(confirmationCode)))
		return 1 // number of results
	}))
	// Set the minimum confirmation code length
	// Takes the minimum number of characters
	L.SetGlobal("SetMinimumConfirmationCodeLength", L.NewFunction(func(L *lua.LState) int {
		length := int(L.ToNumber(1))
		userstate.SetMinimumConfirmationCodeLength(length)
		return 0 // number of results
	}))
	// Generates and returns a unique confirmation code, or an empty string
	// Takes no parameters
	L.SetGlobal("GenerateUniqueConfirmationCode", L.NewFunction(func(L *lua.LState) int {
		confirmationCode, err := userstate.GenerateUniqueConfirmationCode()
		var result lua.LString
		if err != nil {
			result = lua.LString("")
		} else {
			result = lua.LString(confirmationCode)
		}
		L.Push(result)
		return 1 // number of results
	}))
}
