// Package websocket provides Lua functions related to websockets
package websocket

import (
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	log "github.com/sirupsen/logrus"
	//"github.com/yuin/gopher-lua"
	"net/http"
)

// One can not upgrade to websocket on the regular http.ResponseWriter and http.Request arguments
// because they may be `httptest` under the hood, in order to provide better Lua error messages.
// The solution is for the websocket function to create its own server, given two Lua functions
// that can handle reads and writes.
func echoServer(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Errorf("WebSocket UpgradeHTTP error: %s", err)
		// handle error
	}
	// A simple echo server
	go func() {
		defer conn.Close()
		for {
			msg, op, err := wsutil.ReadClientData(conn)
			if err != nil {
				log.Errorf("WebSocket ReadClientData error: %s", err)
				// handle error
			}
			err = wsutil.WriteServerMessage(conn, op, msg)
			if err != nil {
				log.Errorf("WebSocket WriteServerMessage error: %s", err)
				// handle error
			}
		}
	}()
}

//// Load makes functions related to websockets available to the given Lua state
//func Load(L *lua.LState, w http.ResponseWriter, r *http.Request) {
//
//	// Function that upgrades the current handler
//	L.SetGlobal("websocket", L.NewFunction(func(L *lua.LState) int {
//		// Upgrade this ResponseWriter and Request to a WebSocket echo server
//		websocketEchoServer(L, w, r)
//		log.Info("Launched websocket echo server")
//		return 0 // number of results
//	}))
//
//}
