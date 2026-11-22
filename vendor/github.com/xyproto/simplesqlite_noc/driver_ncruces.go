//go:build !(((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64)))

package simplesqlite

import _ "github.com/ncruces/go-sqlite3/driver"

const driverName = "sqlite3"
