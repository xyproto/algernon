//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package simplesqlite

import _ "github.com/liustack/sqlite"

const driverName = "sqlite"
