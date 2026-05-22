//go:build linux || freebsd || windows || netbsd || darwin

package engine

import (
	"net/http"
	"sync/atomic"

	"github.com/quic-go/quic-go/http3"
	"github.com/sirupsen/logrus"
)

const quicEnabled = true

// ListenAndServeQUIC attempts to serve the given http.Handler over QUIC/HTTP3,
// then reports back any errors when done serving.
func (ac *Config) ListenAndServeQUIC(mux http.Handler, justServeRegularHTTP chan bool, servingHTTPS *atomic.Bool) {
	// TODO: * Handle ctrl-c by fetching the quicServer struct and passing it to GenerateShutdownFunction.
	//         This can be done once CloseGracefully in h2quic has been implemented:
	//         https://github.com/lucas-clemente/quic-go/blob/master/h2quic/server.go#L257
	//       * As far as I can tell, this was never implemented. Look into implementing this for github.com/xyproto/quic
	//       * See also: https://github.com/quic-go/quic-go/blob/3cb5f3e104d269768415cce79ddcc5018c79ea92/integrationtests/self/http_shutdown_test.go#L36
	//
	// gracefulServer.ShutdownInitiated = ac.GenerateShutdownFunction(nil, quicServer)
	if err := http3.ListenAndServeTLS(ac.serverAddr, ac.serverCert, ac.serverKey, mux); err != nil {
		logrus.Error("Not serving QUIC after all. Error: ", err)
		logrus.Info("Use the -t flag for serving regular HTTP instead")
		// If QUIC failed (perhaps the key + cert are missing),
		// serve plain HTTP instead
		justServeRegularHTTP <- true
		servingHTTPS.Store(false)
	}
}

// serveQUICPortSetting serves HTTP/3 (QUIC) for an explicit PortSetting entry.
func (ac *Config) serveQUICPortSetting(mux http.Handler, ps PortSetting) {
	// QUIC inherently requires TLS at the transport layer.
	// Even with tls=false in the config, we still need cert/key to establish QUIC connections.
	if err := http3.ListenAndServeTLS(ps.Addr, ac.serverCert, ac.serverKey, mux); err != nil {
		logrus.Errorf("HTTP/3 (QUIC) on %s failed: %v", ps.Addr, err)
	}
}
