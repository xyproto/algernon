TODO
====

Priority
--------

- [ ] Document how to add a custom role (like admin/user/public).

For the next major version
--------------------------

- [ ] Let `HashPassword` return an error instead of panic if bcrypt should fail.
- [ ] Let `NewUserState` return an error instead of the user having to check the Redis connection first.

Maybe
-----

- [ ] Use a more international selection of letters when validating usernames (in `userstate.go`).
- [ ] Look into writing samples for:
  - [ ] [pilu/traffix](https://github.com/pilu/traffic)
  - [ ] [beego](https://github.com/astaxie/beego)
     -  See: [filter.md](http://beego.me/docs/mvc/controller/filter.md)
  - [ ] [gocraft/web](https://github.com/gocraft/web)
  - [ ] [revel](https://github.com/revel/revel)
- [ ]  Look into supporting HTTP basic auth, but only for some paths (see [scoreserver](https://github.com/xyproto/scoreserver)).

