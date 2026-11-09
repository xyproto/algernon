//go:build unix && !solaris

package sqlite3_wrap

import "golang.org/x/sys/unix"

// On Linux, Darwin, and BSDs, PROT_NONE and PROT_READ private anonymous mappings
// do not reserve swap or count against commit limits.
//
// Omitting MAP_NORESERVE preserves Linux's overcommit checks when mprotect later
// makes pages writable.
//
// FreeBSD/AIX/zOS do not define MAP_NORESERVE, and it's ignored on the other BSDs.
const _MAP_PLACEHOLDER = unix.MAP_PRIVATE | unix.MAP_ANON
