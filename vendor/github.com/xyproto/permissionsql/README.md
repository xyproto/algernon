#PermissionSQL [![Build Status](https://travis-ci.org/xyproto/permissionsql.svg?branch=master)](https://travis-ci.org/xyproto/permissionsql) [![GoDoc](https://godoc.org/github.com/xyproto/permissionsql?status.svg)](http://godoc.org/github.com/xyproto/permissionsql)

Middleware for keeping track of users, login states and permissions.

Uses MariaDB/MySQL as a backend. For using [Redis](http://redis.io) as a backend instead, look into [permissions2](https://github.com/xyproto/permissions2).

Background
----------

There were a feature request for [permissions2](https://github.com/xyproto/permissions2) for adding MySQL support.

At first I tried combining the code for Redis database access and SQL database access in the [simpleredis](https://github.com/xyproto/simpleredis) package. I tried interfaces and all sorts of trickery and refactoring, but the result was unsatisfactory, because Redis and SQL databases are so different. However, creating a MariaDB/MySQL version of [simpleredis](https://github.com/xyproto/simpleredis) called [db](https://github.com/xyproto/db) worked out nicely. The [db](https://github.com/xyproto/db) package tries to address the shortcomings of handling UTF-8 strings in MariaDB/MySQL and provide the same functions and behavior as [simpleredis](https://github.com/xyproto/simpleredis), but not with the same performance characteristics.

I recommend using Redis and [permissions2](https://github.com/xyproto/permissions2) instead of this package, if possible.

(For the record, I prefer PostgreSQL over MariaDB/MySQL).


Features and limitations
------------------------

* Uses secure cookies and stores user information in a MariaDB/MySQL database. 
* Suitable for running a local MariaDB/MySQL server, registering/confirming users and managing public/user/admin pages.
* Also supports connecting to remote MariaDB/MySQL servers.
* Supports registration and confirmation via generated confirmation codes.
* Tries to keep things simple.
* Only supports "public", "user" and "admin" permissions out of the box, but offers functionality for implementing more fine grained permissions, if so desired.
* Supports [Negroni](https://github.com/codegangsta/negroni), [Martini](https://github.com/go-martini/martini), [Gin](https://github.com/gin-gonic/gin) and [Macaron](https://github.com/Unknwon/macaron).
* Should also work with other frameworks, since the standard http.HandlerFunc is used everywhere.
* The default permissions can be cleared with the Clear() function.


Example for [Negroni](https://github.com/codegangsta/negroni)
--------------------
~~~ go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/codegangsta/negroni"
	"github.com/xyproto/permissionsql"
)

func main() {
	n := negroni.Classic()
	mux := http.NewServeMux()

	// New permissionsql middleware
	perm, err := permissionsql.New()
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
		fmt.Fprintf(w, "\nTry: /register, /confirm, /remove, /login, /logout, /data, /makeadmin and /admin")
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

	// Enable the permissionsql middleware
	n.Use(perm)

	// Use mux for routing, this goes last
	n.UseHandler(mux)

	// Serve
	n.Run(":3000")
}
~~~

Example for [Martini](https://github.com/go-martini/martini)
--------------------
~~~ go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-martini/martini"
	"github.com/xyproto/permissionsql"
)

func main() {
	m := martini.Classic()

	// New permissionsql middleware
	perm, err := permissionsql.New()
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
		fmt.Fprintf(w, "\nTry: /register, /confirm, /remove, /login, /logout, /data, /makeadmin and /admin")
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

	// Enable the permissionsql middleware
	m.Use(permissionHandler)

	// Serve
	m.Run()
}
~~~

Example for [Gin](https://github.com/gin-gonic/gin)
--------------------
~~~ go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xyproto/permissionsql"
)

func main() {
	g := gin.New()

	// New permissionsql middleware
	perm, err := permissionsql.New()
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
			c.Abort(http.StatusForbidden)
			fmt.Fprint(c.Writer, "Permission denied!")
			return
		}
		// Call the next middleware handler
		c.Next()
	}

	// Logging middleware
	g.Use(gin.Logger())

	// Enable the permissionsql middleware, must come before recovery
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
		msg += fmt.Sprintln("\nTry: /register, /confirm, /remove, /login, /logout, /data, /makeadmin and /admin")
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

Example for [Macaron](https://github.com/Unknwon/macaron)
--------------------
~~~ go
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

	// New permissionsql middleware
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

	// Enable the permissionsql middleware, must come before recovery
	m.Use(permissionHandler)

	// Recovery middleware
	m.Use(macaron.Recovery())

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	m.Get("/", func(ctx *macaron.Context) string {
		msg := ""
		msg += fmt.Sprintf("Has user bob: %v\n", userstate.HasUser("bob"))
		msg += fmt.Sprintf("Logged in on server: %v\n", userstate.IsLoggedIn("bob"))
		msg += fmt.Sprintf("Is confirmed: %v\n", userstate.IsConfirmed("bob"))
		msg += fmt.Sprintf("Username stored in cookies (or blank): %v\n", userstate.Username(ctx.Req.Request))
		msg += fmt.Sprintf("Current user is logged in, has a valid cookie and *user rights*: %v\n", userstate.UserRights(ctx.Req.Request))
		msg += fmt.Sprintf("Current user is logged in, has a valid cookie and *admin rights*: %v\n", userstate.AdminRights(ctx.Req.Request))
		msg += fmt.Sprintln("\nTry: /register, /confirm, /remove, /login, /logout, /data, /makeadmin and /admin")
		return msg
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
~~~


Default permissions
-------------------

* The */admin* path prefix has admin rights by default.
* These path prefixes has user rights by default: */repo* and */data*
* These path prefixes are public by default: */*, */login*, */register*, */style*, */img*, */js*, */favicon.ico*, */robots.txt* and */sitemap_index.xml*

The default permissions can be cleared with the Clear() function.


Password hashing
----------------

* bcrypt is used by default for hashing passwords. sha256 is also supported.
* By default, all new password will be hashed with bcrypt.
* For backwards compatibility, old password hashes with the length of a sha256 hash will be checked with sha256. To disable this behavior, and only ever use bcrypt, add this line: `userstate.SetPasswordAlgo("bcrypt")`

Coding style
------------

* log.Fatal or panic shall only be used for problems that may occur when starting the application, like not being able to connect to the database. The rest of the functions should return errors instead, so that they can be handled.
* The code shall always be formatted with `go fmt`.

Online API Documentation
------------------------

[godoc.org](http://godoc.org/github.com/xyproto/permissionsql)

General information
-------------------

* Version: 2.0
* License: MIT
* Alexander F Rødseth

