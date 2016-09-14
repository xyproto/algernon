TODO
====

Priority
--------

* Write tests for timing attacks (the way it is currently done should be safe,
  but write tests to be sure, now and for future versions).


For the next version
--------------------

* Let HashPassword return an error instead of panic if bcrypt should fail.
* Let NewUserState return an error instead of the user having to check the
  Redis connection first.


Maybe
-----

* Look into writing samples for:
   * https://github.com/pilu/traffic
   * https://github.com/astaxie/beego
        * See: http://beego.me/docs/mvc/controller/filter.md
   * https://github.com/gocraft/web
   * https://github.com/revel/revel
* Look into supporting HTTP basic auth, but only for some paths (see https://github.com/xyproto/scoreserver).
* Write a sample for how to add custom data for a user.
* Write a sample for how to add a custom role (like admin/user/public).
* Use a more international selection of letters when validating usernames (in userstate.go).

