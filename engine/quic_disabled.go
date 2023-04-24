//go:build plan9 || solaris || openbsd || (darwin && !amd64)
// +build plan9 solaris openbsd darwin,!amd64

package engine

import (
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

var quicEnabled = false

// ListanAndServeQUIC is just a placeholder for platforms with QUIC disabled
func (ac *Config) ListenAndServeQUIC(_ http.Handler, _ *sync.Mutex, _ chan bool, _ *bool) {
	log.Error("Not serving QUIC. This Algernon executable was built without QUIC-support.")
}
