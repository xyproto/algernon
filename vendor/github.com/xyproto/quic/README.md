# QUIC aka. HTTP/3

This is a modified version of [quic-go](https://github.com/lucas-clemente/quic-go), to make it build with Go 1.11, Go 1.12 and Go 1.13 (not just "Go 1.13+"), and also GCC 9.2 (`gcc-go`).

Support for "crypto/ed25519" was removed from the TLS 1.3 library, for compatibility with Go 1.11 and `gcc-go`.

* Version: 1.0.2
* License: MIT
