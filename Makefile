.PHONY: clean install install-doc cover test

PROJECT ?= algernon

GOBUILD := go build -mod=vendor -v

#GOEXPERIMENT := greenteagc

# macOS and FreeBSD detection
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  PREFIX ?= /usr/local
  MAKE ?= make
else ifeq ($(UNAME_S),FreeBSD)
  PREFIX ?= /usr/local
  MAKE ?= gmake
else
  PREFIX ?= /usr
  MAKE ?= make
endif

MANDIR ?= $(PREFIX)/share/man/man1
DATADIR ?= $(PREFIX)/share
DOCDIR ?= $(PREFIX)/share/doc

ifneq (,$(wildcard /etc/arch-release))
# Arch Linux
LDFLAGS ?= -Wl,-O2,--sort-common,--as-needed,-z,relro,-z,now
BUILDFLAGS ?= -mod=vendor -buildmode=pie -trimpath -buildvcs=false -ldflags "-s -w -linkmode=external -extldflags $(LDFLAGS)"
else
# Default settings
BUILDFLAGS ?= -mod=vendor -trimpath -ldflags "-s -w" -buildvcs=false
endif

algernon:
	$(GOBUILD) $(BUILDFLAGS)

algernon.1.gz: algernon.1
	gzip -f -k -v algernon.1

install: algernon desktop/mdview
	mkdir -p "$(DESTDIR)$(PREFIX)/bin"
	install -m755 algernon "$(DESTDIR)$(PREFIX)/bin/algernon"
	install -m755 desktop/mdview "$(DESTDIR)$(PREFIX)/bin/mdview"

install-doc: algernon.1.gz welcome.sh samples README.md
	mkdir -p "$(DESTDIR)$(MANDIR)"
	install -m644 algernon.1.gz "$(DESTDIR)$(MANDIR)/algernon.1.gz"
	mkdir -p "$(DESTDIR)$(DATADIR)/algernon"
	cp -r samples "$(DESTDIR)$(DATADIR)/algernon"
	sed 's/\.\/algernon/algernon/g' welcome.sh > welcome_install.sh
	install -m755 welcome_install.sh "$(DESTDIR)$(DATADIR)/algernon/welcome.sh"
	rm -f welcome_install.sh
	mkdir -p "$(DESTDIR)$(DOCDIR)/algernon"
	install -Dm644 README.md "$(DESTDIR)$(DOCDIR)/algernon/README.md"

cover:
	go test -mod=vendor -coverprofile=coverage.out -coverpkg=./... ./...
	go tool cover -func=coverage.out

test:
	go test ./... -mod=vendor -v

clean:
	rm -f algernon algernon.1.gz coverage.out
