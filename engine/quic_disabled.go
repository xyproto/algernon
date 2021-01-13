// +build plan9 solaris openbsd darwin,!amd64

package engine

import (
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

var quicEnabled = false

// ListanAndServeQUIC is just a placeholder for platforms with QUIC disabled
func (ac *Config) ListenAndServeQUIC(mux http.Handler, mut *sync.Mutex, justServeRegularHTTP chan bool, servingHTTPS *bool) {
	log.Error("Not serving QUIC. This Algernon executable was built without QUIC-support.")
}
