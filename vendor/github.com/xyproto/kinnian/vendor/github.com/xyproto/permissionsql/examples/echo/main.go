package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
	"github.com/labstack/echo/middleware"
	"github.com/xyproto/permissionsql"
)

// Convenience function for making it easier to get hold of http.ResponseWriter
func w(c echo.Context) http.ResponseWriter {
	return c.Response().(*standard.Response).ResponseWriter
}

// Convenience function for making it easier to get hold of *http.Request
func req(c echo.Context) *http.Request {
	return c.Request().(*standard.Request).Request
}

func main() {
	e := echo.New()

	// New permissions middleware
	perm, err := permissionsql.New()
	if err != nil {
		log.Fatalln(err)
	}

	// Blank slate, no default permissions
	//perm.Clear()

	// Set up a middleware handler for Echo, with a custom "permission denied" message.
	permissionHandler := echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(c echo.Context) error {
			// Check if the user has the right admin/user rights
			if perm.Rejected(w(c), req(c)) {
				// Deny the request
				return echo.NewHTTPError(http.StatusForbidden, "Permission denied!")
			}
			// Continue the chain of middleware
			return next(c)
		})
	})

	// Logging middleware
	e.Use(middleware.Logger())

	// Enable the permissions middleware, must come before recovery
	e.Use(permissionHandler)

	// Recovery middleware
	e.Use(middleware.Recover())

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	e.Get("/", echo.HandlerFunc(func(c echo.Context) error {
		var buf bytes.Buffer
		b2s := map[bool]string{false: "false", true: "true"}
		buf.WriteString("Has user bob: " + b2s[userstate.HasUser("bob")] + "\n")
		buf.WriteString("Logged in on server: " + b2s[userstate.IsLoggedIn("bob")] + "\n")
		buf.WriteString("Is confirmed: " + b2s[userstate.IsConfirmed("bob")] + "\n")
		buf.WriteString("Username stored in cookies (or blank): " + userstate.Username(req(c)) + "\n")
		buf.WriteString("Current user is logged in, has a valid cookie and *user rights*: " + b2s[userstate.UserRights(req(c))] + "\n")
		buf.WriteString("Current user is logged in, has a valid cookie and *admin rights*: " + b2s[userstate.AdminRights(req(c))] + "\n")
		buf.WriteString("\nTry: /register, /confirm, /remove, /login, /logout, /makeadmin, /clear, /data and /admin")
		return c.String(http.StatusOK, buf.String())
	}))

	e.Get("/register", echo.HandlerFunc(func(c echo.Context) error {
		userstate.AddUser("bob", "hunter1", "bob@zombo.com")
		return c.String(http.StatusOK, fmt.Sprintf("User bob was created: %v\n", userstate.HasUser("bob")))
	}))

	e.Get("/confirm", echo.HandlerFunc(func(c echo.Context) error {
		userstate.MarkConfirmed("bob")
		return c.String(http.StatusOK, fmt.Sprintf("User bob was confirmed: %v\n", userstate.IsConfirmed("bob")))
	}))

	e.Get("/remove", echo.HandlerFunc(func(c echo.Context) error {
		userstate.RemoveUser("bob")
		return c.String(http.StatusOK, fmt.Sprintf("User bob was removed: %v\n", !userstate.HasUser("bob")))
	}))

	e.Get("/login", echo.HandlerFunc(func(c echo.Context) error {
		// Headers will be written, for storing a cookie
		userstate.Login(w(c), "bob")
		return c.String(http.StatusOK, fmt.Sprintf("bob is now logged in: %v\n", userstate.IsLoggedIn("bob")))
	}))

	e.Get("/logout", echo.HandlerFunc(func(c echo.Context) error {
		userstate.Logout("bob")
		return c.String(http.StatusOK, fmt.Sprintf("bob is now logged out: %v\n", !userstate.IsLoggedIn("bob")))
	}))

	e.Get("/makeadmin", echo.HandlerFunc(func(c echo.Context) error {
		userstate.SetAdminStatus("bob")
		return c.String(http.StatusOK, fmt.Sprintf("bob is now administrator: %v\n", userstate.IsAdmin("bob")))
	}))

	e.Get("/clear", echo.HandlerFunc(func(c echo.Context) error {
		userstate.ClearCookie(w(c))
		return c.String(http.StatusOK, "Clearing cookie")
	}))

	e.Get("/data", echo.HandlerFunc(func(c echo.Context) error {
		return c.String(http.StatusOK, "user page that only logged in users must see!")
	}))

	e.Get("/admin", echo.HandlerFunc(func(c echo.Context) error {
		var buf bytes.Buffer
		buf.WriteString("super secret information that only logged in administrators must see!\n\n")
		if usernames, err := userstate.AllUsernames(); err == nil {
			buf.WriteString("list of all users: " + strings.Join(usernames, ", "))
		}
		return c.String(http.StatusOK, buf.String())
	}))

	// Serve
	e.Run(standard.New(":3000"))
}
