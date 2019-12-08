# Plans

- [ ] Use the anti timing-attack from [auth](https://github.com/martini-contrib/auth/).
- [ ] Look into supporting HTTP basic auth, but only for some paths (see [scoreserver](https://github.com/xyproto/scoreserver)).
- [ ] Use a more international selection of letters when validating usernames (in `userstate.go`).
- [ ] Let HashPassword return an error instead of panic if bcrypt should fail (unlikely), for permissions3.

