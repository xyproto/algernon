.PHONY: clean install install-doc

PROJECT ?= algernon

GOBUILD := go build -mod=vendor -v

GOEXPERIMENT := greenteagc

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

UNAME_R ?= $(shell uname -r)
ifneq (,$(findstring arch,$(UNAME_R)))
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
	mkdir -p "$(DESTDIR)$(PREFIX)/usr/share/algernon"
	cp -r samples "$(DESTDIR)$(PREFIX)/usr/share/algernon"
	sed 's/\.\/algernon/algernon/g' welcome.sh > welcome_install.sh
	install -m755 welcome_install.sh "$(DESTDIR)$(PREFIX)/usr/share/algernon/welcome.sh"
	rm -f welcome_install.sh
	mkdir -p "$(DESTDIR)$(PREFIX)/usr/share/doc/algernon"
	install -Dm644 README.md "$(DESTDIR)$(PREFIX)/usr/share/doc/algernon/README.md"

clean:
	rm -f algernon algernon.1.gz
