package alg

import (
	"github.com/xyproto/mime"
)

func (ac *Config) initializeMime() {
	// Read in the mimetype information from the system. Set UTF-8 when setting Content-Type.
	ac.mimereader = mime.New("/etc/mime.types", true)
}
