# Permissions2 [![Build Status](https://travis-ci.org/xyproto/permissions2.svg?branch=master)](https://travis-ci.org/xyproto/permissions2) [![GoDoc](https://godoc.org/github.com/xyproto/permissions2?status.svg)](http://godoc.org/github.com/xyproto/permissions2) [![Report Card](https://img.shields.io/badge/go_report-A+-brightgreen.svg?style=flat)](http://goreportcard.com/report/xyproto/permissions2)

Middleware for keeping track of users, login states and permissions.

## Online API Documentation

[godoc.org](http://godoc.org/github.com/xyproto/permissions2)


## Features and limitations

* Uses secure cookies and stores user information in a Redis database.
* Suitable for running a local Redis server, registering/confirming users and managing public/user/admin pages.
* Also supports connecting to remote Redis servers.
* Does not support SQL databases. For MariaDB/MySQL support, look into [permissionsql](https://github.com/xyproto/permissionsql).
* For Bolt database support (no database host needed, uses a file), look into [permissionbolt](https://github.com/xyproto/permissionbolt).
* For PostgreSQL database support (using the HSTORE feature), look into [pstore](https://github.com/xyproto/pstore).
* Supports registration and confirmation via generated confirmation codes.
* Tries to keep things simple.
* Only supports *public*, *user* and *admin* permissions out of the box, but offers functionality for implementing more fine grained permissions, if so desired.
* The default permissions can be cleared with the `Clear()` function.
* Supports [Negroni](https://github.com/urfave/negroni), [Iris](https://github.com/kataras/iris), [Martini](https://github.com/go-martini/martini), [Gin](https://github.com/gin-gonic/gin), [Goji](https://github.com/zenazn/goji) and plain `net/http`.
* Should also work with other frameworks, since the standard `http.HandlerFunc` is used everywhere.

## Requirements

* Redis >= 2.6.12
* Go >= 1.9

## Examples

There is more information after the examples.

### Example for [Negroni](https://github.com/urfave/negroni)

~~~ go
package main

import (
	"fmt"
	"net/http"
	"strings"
	"log"

	"github.com/urfave/negroni"
	"github.com/xyproto/permissions2"
)

func main() {
	n := negroni.Classic()
	mux := http.NewServeMux()

	// New permissions middleware
	perm, err := permissions.New2()
	if err != nil {
		log.Fatalln(err)
	}

	// Blank slate, no default permissions
	//perm.Clear()

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Has user bob: %v\n", userstate.HasUser("bob"))
		fmt.Fprintf(w, "Logged in on server: %v\n", userstate.IsLoggedIn("bob"))
		fmt.Fprintf(w, "Is confirmed: %v\n", userstate.IsConfirmed("bob"))
		fmt.Fprintf(w, "Username stored in cookies (or blank): %v\n", userstate.Username(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *user rights*: %v\n", userstate.UserRights(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *admin rights*: %v\n", userstate.AdminRights(req))
		fmt.Fprintf(w, "\nTry: /register, /confirm, /remove, /login, /logout, /makeadmin, /clear, /data and /admin")
	})

	mux.HandleFunc("/register", func(w http.ResponseWriter, req *http.Request) {
		userstate.AddUser("bob", "hunter1", "bob@zombo.com")
		fmt.Fprintf(w, "User bob was created: %v\n", userstate.HasUser("bob"))
	})

	mux.HandleFunc("/confirm", func(w http.ResponseWriter, req *http.Request) {
		userstate.MarkConfirmed("bob")
		fmt.Fprintf(w, "User bob was confirmed: %v\n", userstate.IsConfirmed("bob"))
	})

	mux.HandleFunc("/remove", func(w http.ResponseWriter, req *http.Request) {
		userstate.RemoveUser("bob")
		fmt.Fprintf(w, "User bob was removed: %v\n", !userstate.HasUser("bob"))
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, req *http.Request) {
		userstate.Login(w, "bob")
		fmt.Fprintf(w, "bob is now logged in: %v\n", userstate.IsLoggedIn("bob"))
	})

	mux.HandleFunc("/logout", func(w http.ResponseWriter, req *http.Request) {
		userstate.Logout("bob")
		fmt.Fprintf(w, "bob is now logged out: %v\n", !userstate.IsLoggedIn("bob"))
	})

	mux.HandleFunc("/makeadmin", func(w http.ResponseWriter, req *http.Request) {
		userstate.SetAdminStatus("bob")
		fmt.Fprintf(w, "bob is now administrator: %v\n", userstate.IsAdmin("bob"))
	})

	mux.HandleFunc("/clear", func(w http.ResponseWriter, req *http.Request) {
		userstate.ClearCookie(w)
		fmt.Fprintf(w, "Clearing cookie")
	})

	mux.HandleFunc("/data", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "user page that only logged in users must see!")
	})

	mux.HandleFunc("/admin", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "super secret information that only logged in administrators must see!\n\n")
		if usernames, err := userstate.AllUsernames(); err == nil {
			fmt.Fprintf(w, "list of all users: "+strings.Join(usernames, ", "))
		}
	})

	// Custom handler for when permissions are denied
	perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Permission denied!", http.StatusForbidden)
	})

	// Enable the permissions middleware
	n.Use(perm)

	// Use mux for routing, this goes last
	n.UseHandler(mux)

	// Serve
	n.Run(":3000")
}
~~~

### Example for [Iris](https://github.com/kataras/iris)

~~~ go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/kataras/iris"
	"github.com/xyproto/permissions2"
	"github.com/xyproto/pinterface"
)

// PermissionMiddleware returns an new iris.Handler function that
// uses the given perm value to reject or accept HTTP requests.
// `pinterface.IPermissions` is used instead of `*permissions.Permissions`
// in order to be compatible with not only `permissions2`, but also
// other database backends, like `permissionbolt`, which uses BoltDB.
func PermissionMiddleware(perm pinterface.IPermissions) iris.Handler {
	return func(ctx iris.Context) {
		w := ctx.ResponseWriter()
		req := ctx.Request()
		// Check if the user has the right admin/user rights
		if perm.Rejected(w, req) {
			// Stop the request from executing further
			ctx.StopExecution()
			// Let the user know, by calling the custom "permission denied" function
			perm.DenyFunction()(w, req)
			return
		}
		// Serve the next handler if permissions were granted
		ctx.Next()
	}
}

func main() {
	app := iris.New()

	// New permission struct
	perm, err := permissions.New2()
	if err != nil {
		log.Fatalln(err)
	}

	// Blank slate, no default permissions
	//perm.Clear()

	// Enable the permissions middleware
	app.Use(PermissionMiddleware(perm))

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	app.Get("/", func(ctx iris.Context) {
		w := ctx.ResponseWriter()
		req := ctx.Request()
		fmt.Fprintf(w, "Has user bob: %v\n", userstate.HasUser("bob"))
		fmt.Fprintf(w, "Logged in on server: %v\n", userstate.IsLoggedIn("bob"))
		fmt.Fprintf(w, "Is confirmed: %v\n", userstate.IsConfirmed("bob"))
		fmt.Fprintf(w, "Username stored in cookies (or blank): %v\n", userstate.Username(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *user rights*: %v\n", userstate.UserRights(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *admin rights*: %v\n", userstate.AdminRights(req))
		fmt.Fprintf(w, "\nTry: /register, /confirm, /remove, /login, /logout, /makeadmin, /clear, /data and /admin")
	})

	app.Get("/register", func(ctx iris.Context) {
		userstate.AddUser("bob", "hunter1", "bob@zombo.com")
		fmt.Fprintf(ctx.ResponseWriter(), "User bob was created: %v\n", userstate.HasUser("bob"))
	})

	app.Get("/confirm", func(ctx iris.Context) {
		userstate.MarkConfirmed("bob")
		fmt.Fprintf(ctx.ResponseWriter(), "User bob was confirmed: %v\n", userstate.IsConfirmed("bob"))
	})

	app.Get("/remove", func(ctx iris.Context) {
		userstate.RemoveUser("bob")
		fmt.Fprintf(ctx.ResponseWriter(), "User bob was removed: %v\n", !userstate.HasUser("bob"))
	})

	app.Get("/login", func(ctx iris.Context) {
		userstate.Login(ctx.ResponseWriter(), "bob")
		fmt.Fprintf(ctx.ResponseWriter(), "bob is now logged in: %v\n", userstate.IsLoggedIn("bob"))
	})

	app.Get("/logout", func(ctx iris.Context) {
		userstate.Logout("bob")
		fmt.Fprintf(ctx.ResponseWriter(), "bob is now logged out: %v\n", !userstate.IsLoggedIn("bob"))
	})

	app.Get("/makeadmin", func(ctx iris.Context) {
		userstate.SetAdminStatus("bob")
		fmt.Fprintf(ctx.ResponseWriter(), "bob is now administrator: %v\n", userstate.IsAdmin("bob"))
	})

	app.Get("/clear", func(ctx iris.Context) {
		userstate.ClearCookie(ctx.ResponseWriter())
		fmt.Fprintf(ctx.ResponseWriter(), "Clearing cookie")
	})

	app.Get("/data", func(ctx iris.Context) {
		fmt.Fprintf(ctx.ResponseWriter(), "Success!\n\nUser page that only logged in users must see.")
	})

	app.Get("/admin", func(ctx iris.Context) {
		fmt.Fprintf(ctx.ResponseWriter(), "Success!\n\nSuper secret information that only logged in administrators must see.\n\n")
		if usernames, err := userstate.AllUsernames(); err == nil {
			fmt.Fprintf(ctx.ResponseWriter(), "list of all users: "+strings.Join(usernames, ", "))
		}
	})

	// Custom handler for when permissions are denied
	perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Permission denied!", http.StatusForbidden)
	})

	// Serve
	app.Run(iris.Addr(":3000"))
}
~~~


### Example for [Martini](https://github.com/go-martini/martini)

~~~ go
package main

import (
	"fmt"
	"net/http"
	"strings"
	"log"

	"github.com/go-martini/martini"
	"github.com/xyproto/permissions2"
)

func main() {
	m := martini.Classic()

	// New permissions middleware
	perm, err := permissions.New2()
	if err != nil {
		log.Fatalln(err)
	}

	// Blank slate, no default permissions
	//perm.Clear()

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	m.Get("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Has user bob: %v\n", userstate.HasUser("bob"))
		fmt.Fprintf(w, "Logged in on server: %v\n", userstate.IsLoggedIn("bob"))
		fmt.Fprintf(w, "Is confirmed: %v\n", userstate.IsConfirmed("bob"))
		fmt.Fprintf(w, "Username stored in cookies (or blank): %v\n", userstate.Username(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *user rights*: %v\n", userstate.UserRights(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *admin rights*: %v\n", userstate.AdminRights(req))
		fmt.Fprintf(w, "\nTry: /register, /confirm, /remove, /login, /logout, /makeadmin, /clear, /data and /admin")
	})

	m.Get("/register", func(w http.ResponseWriter) {
		userstate.AddUser("bob", "hunter1", "bob@zombo.com")
		fmt.Fprintf(w, "User bob was created: %v\n", userstate.HasUser("bob"))
	})

	m.Get("/confirm", func(w http.ResponseWriter) {
		userstate.MarkConfirmed("bob")
		fmt.Fprintf(w, "User bob was confirmed: %v\n", userstate.IsConfirmed("bob"))
	})

	m.Get("/remove", func(w http.ResponseWriter) {
		userstate.RemoveUser("bob")
		fmt.Fprintf(w, "User bob was removed: %v\n", !userstate.HasUser("bob"))
	})

	m.Get("/login", func(w http.ResponseWriter) {
		userstate.Login(w, "bob")
		fmt.Fprintf(w, "bob is now logged in: %v\n", userstate.IsLoggedIn("bob"))
	})

	m.Get("/logout", func(w http.ResponseWriter) {
		userstate.Logout("bob")
		fmt.Fprintf(w, "bob is now logged out: %v\n", !userstate.IsLoggedIn("bob"))
	})

	m.Get("/makeadmin", func(w http.ResponseWriter) {
		userstate.SetAdminStatus("bob")
		fmt.Fprintf(w, "bob is now administrator: %v\n", userstate.IsAdmin("bob"))
	})

	m.Get("/clear", func(w http.ResponseWriter) {
		userstate.ClearCookie(w)
		fmt.Fprintf(w, "Clearing cookie")
	})

	m.Get("/data", func(w http.ResponseWriter) {
		fmt.Fprintf(w, "user page that only logged in users must see!")
	})

	m.Get("/admin", func(w http.ResponseWriter) {
		fmt.Fprintf(w, "super secret information that only logged in administrators must see!\n\n")
		if usernames, err := userstate.AllUsernames(); err == nil {
			fmt.Fprintf(w, "list of all users: "+strings.Join(usernames, ", "))
		}
	})

	// Set up a middleware handler for Martini, with a custom "permission denied" message.
	permissionHandler := func(w http.ResponseWriter, req *http.Request, c martini.Context) {
		// Check if the user has the right admin/user rights
		if perm.Rejected(w, req) {
			// Deny the request
			http.Error(w, "Permission denied!", http.StatusForbidden)
			// Reject the request by not calling the next handler below
			return
		}
		// Call the next middleware handler
		c.Next()
	}

	// Enable the permissions middleware
	m.Use(permissionHandler)

	// Serve
	m.Run()
}
~~~

### Example for [Gin](https://github.com/gin-gonic/gin)

~~~ go
package main

import (
	"fmt"
	"net/http"
	"strings"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/xyproto/permissions2"
)

func main() {
	g := gin.New()

	// New permissions middleware
	perm, err := permissions.New2()
	if err != nil {
		log.Fatalln(err)
	}

	// Blank slate, no default permissions
	//perm.Clear()

	// Set up a middleware handler for Gin, with a custom "permission denied" message.
	permissionHandler := func(c *gin.Context) {
		// Check if the user has the right admin/user rights
		if perm.Rejected(c.Writer, c.Request) {
			// Deny the request, don't call other middleware handlers
			c.AbortWithStatus(http.StatusForbidden)
			fmt.Fprint(c.Writer, "Permission denied!")
			return
		}
		// Call the next middleware handler
		c.Next()
	}

	// Logging middleware
	g.Use(gin.Logger())

	// Enable the permissions middleware, must come before recovery
	g.Use(permissionHandler)

	// Recovery middleware
	g.Use(gin.Recovery())

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	g.GET("/", func(c *gin.Context) {
		msg := ""
		msg += fmt.Sprintf("Has user bob: %v\n", userstate.HasUser("bob"))
		msg += fmt.Sprintf("Logged in on server: %v\n", userstate.IsLoggedIn("bob"))
		msg += fmt.Sprintf("Is confirmed: %v\n", userstate.IsConfirmed("bob"))
		msg += fmt.Sprintf("Username stored in cookies (or blank): %v\n", userstate.Username(c.Request))
		msg += fmt.Sprintf("Current user is logged in, has a valid cookie and *user rights*: %v\n", userstate.UserRights(c.Request))
		msg += fmt.Sprintf("Current user is logged in, has a valid cookie and *admin rights*: %v\n", userstate.AdminRights(c.Request))
		msg += fmt.Sprintln("\nTry: /register, /confirm, /remove, /login, /logout, /makeadmin, /clear, /data and /admin")
		c.String(http.StatusOK, msg)
	})

	g.GET("/register", func(c *gin.Context) {
		userstate.AddUser("bob", "hunter1", "bob@zombo.com")
		c.String(http.StatusOK, fmt.Sprintf("User bob was created: %v\n", userstate.HasUser("bob")))
	})

	g.GET("/confirm", func(c *gin.Context) {
		userstate.MarkConfirmed("bob")
		c.String(http.StatusOK, fmt.Sprintf("User bob was confirmed: %v\n", userstate.IsConfirmed("bob")))
	})

	g.GET("/remove", func(c *gin.Context) {
		userstate.RemoveUser("bob")
		c.String(http.StatusOK, fmt.Sprintf("User bob was removed: %v\n", !userstate.HasUser("bob")))
	})

	g.GET("/login", func(c *gin.Context) {
		// Headers will be written, for storing a cookie
		userstate.Login(c.Writer, "bob")
		c.String(http.StatusOK, fmt.Sprintf("bob is now logged in: %v\n", userstate.IsLoggedIn("bob")))
	})

	g.GET("/logout", func(c *gin.Context) {
		userstate.Logout("bob")
		c.String(http.StatusOK, fmt.Sprintf("bob is now logged out: %v\n", !userstate.IsLoggedIn("bob")))
	})

	g.GET("/makeadmin", func(c *gin.Context) {
		userstate.SetAdminStatus("bob")
		c.String(http.StatusOK, fmt.Sprintf("bob is now administrator: %v\n", userstate.IsAdmin("bob")))
	})

	g.GET("/clear", func(c *gin.Context) {
		userstate.ClearCookie(c.Writer)
		c.String(http.StatusOK, "Clearing cookie")
	})

	g.GET("/data", func(c *gin.Context) {
		c.String(http.StatusOK, "user page that only logged in users must see!")
	})

	g.GET("/admin", func(c *gin.Context) {
		c.String(http.StatusOK, "super secret information that only logged in administrators must see!\n\n")
		if usernames, err := userstate.AllUsernames(); err == nil {
			c.String(http.StatusOK, "list of all users: "+strings.Join(usernames, ", "))
		}
	})

	// Serve
	g.Run(":3000")
}
~~~

### Example for [Goji](https://github.com/zenazn/goji)

~~~ go
package main

import (
	"fmt"
	"net/http"
	"strings"
	"log"

	"github.com/xyproto/permissions2"
	"github.com/zenazn/goji"
)

func main() {
	// New permissions middleware
	perm, err := permissions.New2()
	if err != nil {
		log.Fatalln(err)
	}

	// Blank slate, no default permissions
	//perm.Clear()

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	goji.Get("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Has user bob: %v\n", userstate.HasUser("bob"))
		fmt.Fprintf(w, "Logged in on server: %v\n", userstate.IsLoggedIn("bob"))
		fmt.Fprintf(w, "Is confirmed: %v\n", userstate.IsConfirmed("bob"))
		fmt.Fprintf(w, "Username stored in cookies (or blank): %v\n", userstate.Username(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *user rights*: %v\n", userstate.UserRights(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *admin rights*: %v\n", userstate.AdminRights(req))
		fmt.Fprintf(w, "\nTry: /register, /confirm, /remove, /login, /logout, /makeadmin, /clear, /data and /admin")
	})

	goji.Get("/register", func(w http.ResponseWriter, req *http.Request) {
		userstate.AddUser("bob", "hunter1", "bob@zombo.com")
		fmt.Fprintf(w, "User bob was created: %v\n", userstate.HasUser("bob"))
	})

	goji.Get("/confirm", func(w http.ResponseWriter, req *http.Request) {
		userstate.MarkConfirmed("bob")
		fmt.Fprintf(w, "User bob was confirmed: %v\n", userstate.IsConfirmed("bob"))
	})

	goji.Get("/remove", func(w http.ResponseWriter, req *http.Request) {
		userstate.RemoveUser("bob")
		fmt.Fprintf(w, "User bob was removed: %v\n", !userstate.HasUser("bob"))
	})

	goji.Get("/login", func(w http.ResponseWriter, req *http.Request) {
		userstate.Login(w, "bob")
		fmt.Fprintf(w, "bob is now logged in: %v\n", userstate.IsLoggedIn("bob"))
	})

	goji.Get("/logout", func(w http.ResponseWriter, req *http.Request) {
		userstate.Logout("bob")
		fmt.Fprintf(w, "bob is now logged out: %v\n", !userstate.IsLoggedIn("bob"))
	})

	goji.Get("/makeadmin", func(w http.ResponseWriter, req *http.Request) {
		userstate.SetAdminStatus("bob")
		fmt.Fprintf(w, "bob is now administrator: %v\n", userstate.IsAdmin("bob"))
	})

	goji.Get("/clear", func(w http.ResponseWriter, req *http.Request) {
		userstate.ClearCookie(w)
		fmt.Fprintf(w, "Clearing cookie")
	})

	goji.Get("/data", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "user page that only logged in users must see!")
	})

	goji.Get("/admin", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "super secret information that only logged in administrators must see!\n\n")
		if usernames, err := userstate.AllUsernames(); err == nil {
			fmt.Fprintf(w, "list of all users: "+strings.Join(usernames, ", "))
		}
	})

	// Custom "permissions denied" message
	perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Permission denied!", http.StatusForbidden)
	})

	// Permissions middleware for Goji
	permissionHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Check if the user has the right admin/user rights
			if perm.Rejected(w, req) {
				// Deny the request
				perm.DenyFunction()(w, req)
				return
			}
			// Serve the requested page
			next.ServeHTTP(w, req)
		})
	}

	// Enable the permissions middleware
	goji.Use(permissionHandler)

	// Goji will listen to port 8000 by default
	goji.Serve()
}
~~~

### Example for just `net/http`

~~~ go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/xyproto/permissions2"
	"github.com/xyproto/pinterface"
)

type permissionHandler struct {
	// perm is a Permissions structure that can be used to deny requests
	// and acquire the UserState. By using `pinterface.IPermissions` instead
	// of `*permissions.Permissions`, the code is compatible with not only
	// `permissions2`, but also other modules that uses other database
	// backends, like `permissionbolt` which uses Bolt.
	perm pinterface.IPermissions

	// The HTTP multiplexer
	mux *http.ServeMux
}

// Implement the ServeHTTP method to make a permissionHandler a http.Handler
func (ph *permissionHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Check if the user has the right admin/user rights
	if ph.perm.Rejected(w, req) {
		// Let the user know, by calling the custom "permission denied" function
		ph.perm.DenyFunction()(w, req)
		// Reject the request
		return
	}
	// Serve the requested page if permissions were granted
	ph.mux.ServeHTTP(w, req)
}

func main() {
	mux := http.NewServeMux()

	// New permissions middleware
	perm, err := permissions.New2()
	if err != nil {
		log.Fatalln(err)
	}

	// Blank slate, no default permissions
	//perm.Clear()

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Has user bob: %v\n", userstate.HasUser("bob"))
		fmt.Fprintf(w, "Logged in on server: %v\n", userstate.IsLoggedIn("bob"))
		fmt.Fprintf(w, "Is confirmed: %v\n", userstate.IsConfirmed("bob"))
		fmt.Fprintf(w, "Username stored in cookies (or blank): %v\n", userstate.Username(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *user rights*: %v\n", userstate.UserRights(req))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *admin rights*: %v\n", userstate.AdminRights(req))
		fmt.Fprintf(w, "\nTry: /register, /confirm, /remove, /login, /logout, /makeadmin, /clear, /data and /admin")
	})

	mux.HandleFunc("/register", func(w http.ResponseWriter, req *http.Request) {
		userstate.AddUser("bob", "hunter1", "bob@zombo.com")
		fmt.Fprintf(w, "User bob was created: %v\n", userstate.HasUser("bob"))
	})

	mux.HandleFunc("/confirm", func(w http.ResponseWriter, req *http.Request) {
		userstate.MarkConfirmed("bob")
		fmt.Fprintf(w, "User bob was confirmed: %v\n", userstate.IsConfirmed("bob"))
	})

	mux.HandleFunc("/remove", func(w http.ResponseWriter, req *http.Request) {
		userstate.RemoveUser("bob")
		fmt.Fprintf(w, "User bob was removed: %v\n", !userstate.HasUser("bob"))
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, req *http.Request) {
		userstate.Login(w, "bob")
		fmt.Fprintf(w, "bob is now logged in: %v\n", userstate.IsLoggedIn("bob"))
	})

	mux.HandleFunc("/logout", func(w http.ResponseWriter, req *http.Request) {
		userstate.Logout("bob")
		fmt.Fprintf(w, "bob is now logged out: %v\n", !userstate.IsLoggedIn("bob"))
	})

	mux.HandleFunc("/makeadmin", func(w http.ResponseWriter, req *http.Request) {
		userstate.SetAdminStatus("bob")
		fmt.Fprintf(w, "bob is now administrator: %v\n", userstate.IsAdmin("bob"))
	})

	mux.HandleFunc("/clear", func(w http.ResponseWriter, req *http.Request) {
		userstate.ClearCookie(w)
		fmt.Fprintf(w, "Clearing cookie")
	})

	mux.HandleFunc("/data", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "user page that only logged in users must see!")
	})

	mux.HandleFunc("/admin", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "super secret information that only logged in administrators must see!\n\n")
		if usernames, err := userstate.AllUsernames(); err == nil {
			fmt.Fprintf(w, "list of all users: "+strings.Join(usernames, ", "))
		}
	})

	// Custom handler for when permissions are denied
	perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Permission denied!", http.StatusForbidden)
	})

	// Configure the HTTP server and permissionHandler struct
	s := &http.Server{
		Addr:           ":3000",
		Handler:        &permissionHandler{perm, mux},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Println("Listening for requests on port 3000")

	// Start listening
	log.Fatal(s.ListenAndServe())
}
~~~

## Default permissions

* Visiting the */admin* path prefix requires the user to be logged in with admin rights, by default.
* These path prefixes requires the user to be logged in, by default: */repo* and */data*
* These path prefixes are public by default: */*, */login*, */register*, */style*, */img*, */js*, */favicon.ico*, */robots.txt* and */sitemap_index.xml*

The default permissions can be cleared with the `Clear()` function.


## Password hashing

* bcrypt is used by default for hashing passwords. sha256 is also supported.
* By default, all new password will be hashed with bcrypt.
* For backwards compatibility, old password hashes with the length of a sha256 hash will be checked with sha256. To disable this behavior, and only ever use bcrypt, add this line: `userstate.SetPasswordAlgo("bcrypt")`


## Coding style

* The code shall always be formatted with `go fmt`.


## Setting and getting properties for users

* Setting a property:

```
username := "bob"
propertyName := "clever"
propertyValue := "yes"

userstate.Users().Set(username, propertyName, propertyValue)
```

* Getting a property:

```
username := "bob"
propertyName := "clever"
propertyValue, err := userstate.Users().Get(username, propertyName)
if err != nil {
	log.Print(err)
	return err
}
fmt.Printf("%s is %s: %s\n", username, propertyName, propertyValue)
```

## Passing userstate between functions, files and to other Go packages

Using the `pinterface.IUserState` interface (from the [pinterface](https://github.com/xyproto/pinterface) package) makes it possible to pass UserState structs between functions, also in other packages. By using this, it is possible to seamlessly change the database backend from, for instance, Redis ([permissions2](https://github.com/xyproto/permissions2)) to BoltDB ([permissionbolt](https://github.com/xyproto/permissionbolt)).

[pstore](https://github.com/xyproto/pstore), [permissionsql](https://github.com/xyproto/permissionsql), [permissionbolt](https://github.com/xyproto/permissionbolt) and [permissions2](https://github.com/xyproto/permissions2) are interchangeable.


## Retrieving the underlying Redis database

Here is a short example application for retrieving the underlying Redis pool and connection:

```go
package main

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/xyproto/permissions2"
)

func main() {
	perm, err := permissions.New2()
	if err != nil {
		fmt.Println("Could not open Redis database")
		return
	}
	ustate := perm.UserState()

	// A bit of checking is needed, since the database backend is interchangeable
	pustate, ok := ustate.(*permissions.UserState)
	if !ok {
		fmt.Println("Not using the Redis database backend")
		return
	}

	// Convert from a simpleredis.ConnectionPool to a redis.Pool
	redisPool := redis.Pool(*pustate.Pool())
	fmt.Printf("Redis pool: %v (%T)\n", redisPool, redisPool)

	// Get the Redis connection as well
	redisConnection := redisPool.Get()
	fmt.Printf("Redis connection: %v (%T)\n", redisConnection, redisConnection)
}
```

Note that the `redigo` repository was recently moved to `https://github.com/gomodule/redigo`. The above code will not work if you use the old redigo package.

## General information

* Version: 2.6.0
* License: MIT
* Alexander F Rødseth &lt;xyproto@archlinux.org&gt;

