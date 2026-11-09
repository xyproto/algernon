package sqlite3_wrap

import "golang.org/x/sys/unix"

// Solaris/illumos does not overcommit: PROT_NONE and PROT_READ private anonymous mappings
// reserve swap unless MAP_NORESERVE is set.
const _MAP_PLACEHOLDER = unix.MAP_PRIVATE | unix.MAP_ANON | unix.MAP_NORESERVE
