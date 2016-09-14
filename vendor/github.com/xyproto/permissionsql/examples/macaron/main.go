package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Unknwon/macaron"
	"github.com/xyproto/permissionsql"
)

func main() {
	m := macaron.Classic()

	// New permissions middleware
	perm, err := permissionsql.New()
	if err != nil {
		log.Fatalln(err)
	}

	// Blank slate, no default permissions
	//perm.Clear()

	// Logging middleware
	m.Use(macaron.Logger())

	// Renderer middleware
	m.Use(macaron.Renderer())

	// Set up a middleware handler for Macaron, with a custom "permission denied" message.
	permissionHandler := func(ctx *macaron.Context) {
		// Check if the user has the right admin/user rights
		if perm.Rejected(ctx.Resp, ctx.Req.Request) {
			fmt.Fprintf(ctx.Resp, "Permission denied!")
			// Deny the request
			ctx.Error(http.StatusForbidden)
			// Don't call other middleware handlers
			return
		}
		// Call the next middleware handler
		ctx.Next()
	}

	// Enable the permissions middleware, must come before recovery
	m.Use(permissionHandler)

	// Recovery middleware
	m.Use(macaron.Recovery())

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	m.Get("/", func(ctx *macaron.Context) (msg string) {
		msg += fmt.Sprintf("Has user bob: %v\n", userstate.HasUser("bob"))
		msg += fmt.Sprintf("Logged in on server: %v\n", userstate.IsLoggedIn("bob"))
		msg += fmt.Sprintf("Is confirmed: %v\n", userstate.IsConfirmed("bob"))
		msg += fmt.Sprintf("Username stored in cookies (or blank): %v\n", userstate.Username(ctx.Req.Request))
		msg += fmt.Sprintf("Current user is logged in, has a valid cookie and *user rights*: %v\n", userstate.UserRights(ctx.Req.Request))
		msg += fmt.Sprintf("Current user is logged in, has a valid cookie and *admin rights*: %v\n", userstate.AdminRights(ctx.Req.Request))
		msg += fmt.Sprintln("\nTry: /register, /confirm, /remove, /login, /logout, /data, /makeadmin and /admin")
		return // msg
	})

	m.Get("/register", func(ctx *macaron.Context) string {
		userstate.AddUser("bob", "hunter1", "bob@zombo.com")
		return fmt.Sprintf("User bob was created: %v\n", userstate.HasUser("bob"))
	})

	m.Get("/confirm", func(ctx *macaron.Context) string {
		userstate.MarkConfirmed("bob")
		return fmt.Sprintf("User bob was confirmed: %v\n", userstate.IsConfirmed("bob"))
	})

	m.Get("/remove", func(ctx *macaron.Context) string {
		userstate.RemoveUser("bob")
		return fmt.Sprintf("User bob was removed: %v\n", !userstate.HasUser("bob"))
	})

	m.Get("/login", func(ctx *macaron.Context) string {
		// Headers will be written, for storing a cookie
		userstate.Login(ctx.Resp, "bob")
		return fmt.Sprintf("bob is now logged in: %v\n", userstate.IsLoggedIn("bob"))
	})

	m.Get("/logout", func(ctx *macaron.Context) string {
		userstate.Logout("bob")
		return fmt.Sprintf("bob is now logged out: %v\n", !userstate.IsLoggedIn("bob"))
	})

	m.Get("/makeadmin", func(ctx *macaron.Context) string {
		userstate.SetAdminStatus("bob")
		return fmt.Sprintf("bob is now administrator: %v\n", userstate.IsAdmin("bob"))
	})

	m.Get("/data", func(ctx *macaron.Context) string {
		return "user page that only logged in users must see!"
	})

	m.Get("/admin", func(ctx *macaron.Context) {
		fmt.Fprintf(ctx.Resp, "super secret information that only logged in administrators must see!\n\n")
		if usernames, err := userstate.AllUsernames(); err == nil {
			fmt.Fprintf(ctx.Resp, "list of all users: "+strings.Join(usernames, ", "))
		}
	})

	// Serve
	m.Run(3000)
}
