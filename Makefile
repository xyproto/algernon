.PHONY: clean

PREFIX ?= /usr
MANDIR ?= "$(PREFIX)/share/man/man1"
GOBUILD := $(shell test $$(go version | tr ' ' '\n' | head -3 | tail -1 | tr '.' '\n' | tail -1) -le 12 && echo GO111MODULES=on go build -v || echo go build -mod=vendor -v)

algernon:
	@go version | grep -q 'go version go1.14' && $(GOBUILD) \
      || echo 'Unfortunately, only Go 1.14 is supported right now'

algernon.1.gz: algernon.1
	gzip -f -k -v algernon.1

install: algernon desktop/mdview
	install -Dm755 algernon "$(DESTDIR)$(PREFIX)/bin/algernon"
	install -Dm755 desktop/mdview "$(DESTDIR)$(PREFIX)/bin/mdview"

install-doc: algernon.1.gz welcome.sh samples README.md
	install -Dm644 algernon.1.gz "$(DESTDIR)$(MANDIR)/algernon.1.gz"
	install -d "$(DESTDIR)$(PREFIX)/usr/share/algernon"
	cp -r samples "$(DESTDIR)$(PREFIX)/usr/share/algernon"
	sed 's/\.\/algernon/algernon/g' welcome.sh > welcome_install.sh
	install -Dm755 welcome_install.sh "$(DESTDIR)$(PREFIX)/usr/share/algernon/welcome.sh"
	rm -f welcome_install.sh
	install -Dm644 README.md "$(DESTDIR)$(PREFIX)/usr/share/doc/algernon/README.md"

clean:
	rm -f algernon algernon.1.gz
