.PHONY: all algernon devinstall race clean

all: algernon

algernon:
	go build

race:
	go run -race basic.go flags.go handlers.go hashmap.go keyvalue.go list.go logo_unix.go lua.go luapool.go main.go prettyerror.go rendering.go servelua.go serverconf.go set.go userstate.go utils.go -httponly

devinstall: algernon
	sudo install -Dm755 algernon /usr/bin/algernon

clean:
	rm -f algernon

