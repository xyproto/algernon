package engine

import (
	"reflect"
	"testing"
)

// exists returns a predicate that reports true for the listed paths.
// Used by classifyPositionalArgs in place of touching the real file system.
func exists(paths ...string) func(string) bool {
	set := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		set[p] = struct{}{}
	}
	return func(p string) bool {
		_, ok := set[p]
		return ok
	}
}

func TestClassifyPositionalArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		existing []string
		want     positionalArgs
	}{
		{
			name: "no args",
			args: nil,
			want: positionalArgs{},
		},
		{
			name:     "directory only",
			args:     []string{"mydir"},
			existing: []string{"mydir"},
			want: positionalArgs{
				ServerDirOrFilename: "mydir",
				ServerDirSet:        true,
			},
		},
		{
			name:     "trailing slash is stripped from directory",
			args:     []string{"mydir/"},
			existing: []string{"mydir/"},
			want: positionalArgs{
				ServerDirOrFilename: "mydir",
				ServerDirSet:        true,
			},
		},
		{
			name: "bare port is treated as web addr",
			args: []string{"8080"},
			want: positionalArgs{
				ServerAddr: ":8080",
			},
		},
		{
			name: "host:port address",
			args: []string{"127.0.0.1:4000"},
			want: positionalArgs{
				ServerAddr: "127.0.0.1:4000",
			},
		},
		{
			name: "IPv6 address",
			args: []string{"[::1]:443"},
			want: positionalArgs{
				ServerAddr: "[::1]:443",
			},
		},
		{
			name: "redis address is recognised",
			args: []string{"127.0.0.1:6379"},
			want: positionalArgs{
				RedisAddr:         "127.0.0.1:6379",
				RedisAddrFromArgs: true,
			},
		},
		{
			name: "cert and key files fall through to Remaining",
			args: []string{"cert.pem", "key.pem"},
			want: positionalArgs{
				Remaining: []string{"cert.pem", "key.pem"},
			},
		},
		{
			name:     "dir, addr and redis together in any order",
			args:     []string{":4000", "srv", "127.0.0.1:6379"},
			existing: []string{"srv"},
			want: positionalArgs{
				ServerDirOrFilename: "srv",
				ServerDirSet:        true,
				ServerAddr:          ":4000",
				RedisAddr:           "127.0.0.1:6379",
				RedisAddrFromArgs:   true,
			},
		},
		{
			name: "redis db index after redis addr goes into Remaining",
			args: []string{"127.0.0.1:6379", "2"},
			want: positionalArgs{
				RedisAddr:         "127.0.0.1:6379",
				RedisAddrFromArgs: true,
				Remaining:         []string{"2"},
			},
		},
		{
			name: "web-looking port is preferred as addr, not redis db index",
			args: []string{"127.0.0.1:6379", "8080"},
			want: positionalArgs{
				ServerAddr:        ":8080",
				RedisAddr:         "127.0.0.1:6379",
				RedisAddrFromArgs: true,
			},
		},
		{
			name: "unknown token falls through",
			args: []string{"nonsense"},
			want: positionalArgs{
				Remaining: []string{"nonsense"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyPositionalArgs(tt.args, exists(tt.existing...))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("classifyPositionalArgs(%v):\n got  %+v\n want %+v", tt.args, got, tt.want)
			}
		})
	}
}

// TestClassifyPositionalArgsNilPredicate ensures a nil pathExists predicate
// does not panic and simply prevents ServerDir detection.
func TestClassifyPositionalArgsNilPredicate(t *testing.T) {
	got := classifyPositionalArgs([]string{"somepath"}, nil)
	want := positionalArgs{Remaining: []string{"somepath"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}
