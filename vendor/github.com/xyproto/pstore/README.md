# pstore

[![Build Status](https://travis-ci.com/xyproto/pstore.svg?branch=master)](https://travis-ci.com/xyproto/pstore)
[![GoDoc](https://godoc.org/github.com/xyproto/pstore?status.svg)](http://godoc.org/github.com/xyproto/pstore)
[![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/pstore)](https://goreportcard.com/report/github.com/xyproto/pstore)

Middleware for keeping track of users, login states and permissions.

Uses [PostgreSQL](https://postgresql.org) for the database. For using [Redis](http://redis.io) as a backend instead, look into [permissions2](https://github.com/xyproto/permissions2).
There is also a [BoltDB](https://github.com/xyproto/permissionbolt) and [MariaDB/MySQL](https://github.com/xyproto/permissionsql) backend. They are interchangeable.

Online API Documentation
------------------------

[godoc.org](https://godoc.org/github.com/xyproto/pstore)

Connecting
----------

For connecting to a PostgreSQL host that is running locally, the `New` function can be used. For connecting to a remote server, the `NewWithDSN` function can be used.


Features and limitations
------------------------

* Uses secure cookies and stores user information in a PostgreSQL database.
* Suitable for running a local PostgreSQL server, registering/confirming users and managing public/user/admin pages.
* Also supports connecting to remote PostgreSQL servers.
* Supports registration and confirmation via generated confirmation codes.
* Tries to keep things simple.
* Only supports "public", "user" and "admin" permissions out of the box, but offers functionality for implementing more fine grained permissions, if so desired.
* Supports [Negroni](https://github.com/codegangsta/negroni), [Martini](https://github.com/go-martini/martini), [Gin](https://github.com/gin-gonic/gin), [Goji](https://github.com/zenazn/goji) and plain `net/http`.
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
	"github.com/xyproto/pstore"
)

func main() {
	n := negroni.Classic()
	mux := http.NewServeMux()

	// New pstore middleware
	perm, err := pstore.New()
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

	// Enable the pstore middleware
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
	"github.com/xyproto/pstore"
)

func main() {
	m := martini.Classic()

	// New pstore middleware
	perm, err := pstore.New()
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

	// Enable the pstore middleware
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
	"github.com/xyproto/pstore"
)

func main() {
	g := gin.New()

	// New pstore middleware
	perm, err := pstore.New()
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

	// Enable the pstore middleware, must come before recovery
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

Example for [Goji](https://github.com/zenazn/goji)
--------------------
~~~ go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/xyproto/pstore"
	"github.com/zenazn/goji"
)

func main() {
	// New permissions middleware
	perm, err := pstore.New()
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

Example for just `net/http`
--------------------

~~~ go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/xyproto/pstore"
	"github.com/xyproto/pinterface"
)

type permissionHandler struct {
	// perm is a Permissions structure that can be used to deny requests
	// and acquire the UserState. By using `pinterface.IPermissions` instead
	// of `*pstore.Permissions`, the code is compatible with not only
	// `pstore`, but also other modules that uses other database
	// backends, like `permissions2` which uses Redis.
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
		// Reject the request by not calling the next handler below
		return
	}
	// Serve the requested page if permissions were granted
	ph.mux.ServeHTTP(w, req)
}

func main() {
	mux := http.NewServeMux()

	// New pstore middleware
	perm, err := pstore.New()
	if err != nil {
		log.Fatal("Could not open Bolt database")
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
	s.ListenAndServe()
}
~~~


Default permissions
-------------------

* The */admin* path prefix has admin rights by default.
* These path prefixes has user rights by default: */repo* and */data*
* These path prefixes are public by default: */*, */login*, */register*, */style*, */img*, */js*, */favicon.ico*, */robots.txt* and */sitemap_index.xml*

The default permissions can be cleared with the `Clear()` function.

Coding style
------------

* The code shall always be formatted with `go fmt`.

Password hashing
----------------

* bcrypt is used by default for hashing passwords. sha256 is also supported.
* By default, all new password will be hashed with bcrypt.
* For backwards compatibility, old password hashes with the length of a sha256 hash will be checked with sha256. To disable this behavior, and only ever use bcrypt, add this line: `userstate.SetPasswordAlgo("bcrypt")`

Passing userstate between functions, files and to other Go packages
-------------------------------------------------------------------

Using the `*pinterface.IUserState` type (from the [pinterface](https://github.com/xyproto/pinterface) package) makes it possible to pass UserState structs between functions, also in other packages. By using this interface, it is possible to seamlessly change the database backend from, for instance, Redis ([permissions2](https://github.com/xyproto/permissions2)) to BoltDB ([permissionbolt](https://github.com/xyproto/permissionbolt)).

[pstore](https://github.com/xyproto/pstore), [permissionsql](https://github.com/xyproto/permissionsql), [permissionbolt](https://github.com/xyproto/permissionbolt) and [permissions2](https://github.com/xyproto/permissions2) are interchangeable.


Retrieving the underlying PostgreSQL database
---------------------------------------------

Here is a short example application for retrieving the underlying PostgreSQL database:

```go
package main

import (
	"fmt"
	"github.com/xyproto/pstore"
	"github.com/xyproto/simplehstore"
)

func main() {
	perm, err := pstore.New()
	if err != nil {
		fmt.Println("Could not open database")
		return
	}
	ustate := perm.UserState()

	// A bit of checking is needed, since the database backend is interchangeable
	if pustate, ok := ustate.(*pstore.UserState); ok {
		if host, ok := pustate.Host().(*simplehstore.Host); ok {
			db := host.Database()
			fmt.Printf("PostgreSQL database: %v (%T)\n", db, db)
		}
	} else {
		fmt.Println("Not using the PostgreSQL database backend")
	}
}
```


General information
-------------------

* Version: 3.0.0
* License: MIT
* Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
