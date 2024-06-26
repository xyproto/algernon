//go:build plan9 || solaris || openbsd
// +build plan9 solaris openbsd

package engine

import (
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

const quicEnabled = false

// ListanAndServeQUIC is just a placeholder for platforms with QUIC disabled
func (ac *Config) ListenAndServeQUIC(_ http.Handler, _ *sync.Mutex, _ chan bool, _ *bool) {
	logrus.Error("Not serving QUIC. This Algernon executable was built without QUIC-support.")
}
