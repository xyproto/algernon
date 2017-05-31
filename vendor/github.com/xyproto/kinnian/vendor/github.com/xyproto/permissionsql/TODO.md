TODO
====

* Add ClearCookie to the examples, like for permissions2 and permissionbolt
* Use the anti timing-attack from martini-contrib/auth/.
* Look into supporting HTTP basic auth, but only for some paths (see xyproto/scoreserver)
* Add custom roles for permissions3
* Decouple the database backend for permissions3 (and add sqlite3 support)
* Use a more international selection of letters when validating usernames (in userstate.go)
* Let HashPassword return an error instead of panic if bcrypt should fail, for permissions3

