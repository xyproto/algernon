TODO
====

* Use the anti timing-attack from martini-contrib/auth/.
* Look into supporting HTTP basic auth, but only for some paths (see xyproto/scoreserver)
* Add custom roles for permissions3
* Use a more international selection of letters when validating usernames (in userstate.go)
* Let HashPassword return an error instead of panic if bcrypt should fail, for permissions3

