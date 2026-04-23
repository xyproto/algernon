package engine

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// positionalArgs is the result of classifying Algernon's free-form positional arguments
type positionalArgs struct {
	// ServerDirOrFilename is an existing directory or file given on the command line
	ServerDirOrFilename string
	ServerDirSet        bool

	// ServerAddr is set from a "host:port", ":port" or bare web-port argument
	ServerAddr string

	// RedisAddr is set when an argument looks like a Redis address (":6379" or ":6380")
	RedisAddr         string
	RedisAddrFromArgs bool

	// Remaining holds unclassified arguments, in the legacy order: cert, key, redis address, redis db index
	Remaining []string
}

// classifyPositionalArgs splits free-form positional arguments into the categories Algernon
// historically supported. If pathExists is nil, the serverDir check is skipped.
func classifyPositionalArgs(args []string, pathExists func(string) bool) positionalArgs {
	if pathExists == nil {
		pathExists = func(string) bool { return false }
	}
	looksLikeCertOrKey := func(arg string) bool {
		switch strings.ToLower(filepath.Ext(arg)) {
		case ".pem", ".crt", ".cer", ".key":
			return true
		default:
			return false
		}
	}
	looksLikeRedisAddr := func(arg string) bool {
		return strings.HasSuffix(arg, ":6379") || strings.HasSuffix(arg, ":6380")
	}
	looksLikeWebPort := func(port string) bool {
		return port == "80" || port == "8080" || strings.HasSuffix(port, "000")
	}

	var result positionalArgs
	for _, arg := range args {
		if looksLikeCertOrKey(arg) {
			result.Remaining = append(result.Remaining, arg)
			continue
		}
		if !result.ServerDirSet && pathExists(arg) {
			if strings.HasSuffix(arg, string(os.PathSeparator)) {
				result.ServerDirOrFilename = arg[:len(arg)-1]
			} else {
				result.ServerDirOrFilename = arg
			}
			result.ServerDirSet = true
			continue
		}
		if looksLikeRedisAddr(arg) {
			result.RedisAddr = arg
			result.RedisAddrFromArgs = true
			continue
		}
		// Check if this looks like a host:port address (handles IPv4, IPv6, and port-only)
		if _, _, err := net.SplitHostPort(arg); err == nil {
			// Valid host:port format, assume this is the web server address
			result.ServerAddr = arg
			continue
		}
		if _, err := strconv.Atoi(arg); err == nil { // no error
			// If in doubt, assume this is the web server port.
			if looksLikeWebPort(arg) || !result.RedisAddrFromArgs {
				result.ServerAddr = net.JoinHostPort("", arg)
			} else {
				result.Remaining = append(result.Remaining, arg)
			}
			continue
		}
		result.Remaining = append(result.Remaining, arg)
	}
	return result
}
