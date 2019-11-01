package utils

import "github.com/xyproto/quic/internal/protocol"

// ByteInterval is an interval from one ByteCount to the other
type ByteInterval struct {
	Start protocol.ByteCount
	End   protocol.ByteCount
}
