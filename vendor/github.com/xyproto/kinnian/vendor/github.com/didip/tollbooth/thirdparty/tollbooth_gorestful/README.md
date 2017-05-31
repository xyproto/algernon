## tollbooth_gorestful

Middleware for [go-restful](https://github.com/emicklei/go-restful)

Import package `thirdparty/tollbooth_gorestful` and use `tollbooth_gorestful.LimitHandler` to which your own handler can be passed

## Five Minutes Tutorial

```
package resources

import (
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/thirdparty/tollbooth_gorestful"
	"github.com/emicklei/go-restful"
)

type User struct {
	ID     string  `json:"id"`
	Email  string  `json:"email"`
}

func (u *User) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/users").Doc("Manage Users").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/{id}").To(tollbooth_gorestful.LimitHandler(u.GetUser, tollbooth.NewLimiter(3, time.Minute))).
		// docs
		Doc("get a user").
		Operation("GetUser").
		Param(ws.PathParameter("id", "identifier of the user").DataType("string")).
		Writes(User{}))

	container.Add(ws)
}
```
