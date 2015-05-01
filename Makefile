.PHONY: all algernon devinstall clean

all: algernon

algernon:
	go build

devinstall: algernon
	sudo install -Dm755 algernon /usr/bin/algernon

clean:
	rm -f algernon

