package handshake

//go:generate sh -c "../mockgen_internal.sh handshake mock_handshake_runner_test.go github.com/xyproto/quic/internal/handshake handshakeRunner"
//go:generate sh -c "mockgen -package handshake crypto/tls ClientSessionCache > mock_client_session_cache_test.go"
