TODO
====

* Look into writing examples for:
   * https://github.com/pilu/traffic
   * https://github.com/astaxie/beego
        * See: http://beego.me/docs/mvc/controller/filter.md
   * https://github.com/gocraft/web
   * https://github.com/revel/revel
* Look into supporting HTTP basic auth, but only for some paths (see https://github.com/xyproto/scoreserver).
* Write tests for timing attacks.
* Write an example for how to add custom data for a user.
* Write an example for how to add a custom role (like admin/user/public).
* Use a more international selection of letters when validating usernames (in userstate.go).
* Let HashPassword return an error instead of panic if bcrypt should fail, for permissions3.
