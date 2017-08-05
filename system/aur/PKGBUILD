# Maintainer: Alexander F Rødseth <xyproto@archlinux.org>

pkgname=algernon
pkgver=1.5.1
pkgrel=1
pkgdesc='Pure Go web server with Lua, Markdown, HTTP/2 and template support'
arch=('x86_64' 'i686')
url='http://algernon.roboticoverlords.org/'
license=('MIT')
makedepends=('go' 'git' 'glide' 'setconf')
optdepends=('redis: For using the Redis database backend'
            'mariadb: For using the MariaDB/MySQL database backend'
            'postgresql: For using the PostgreSQL database backend')
backup=('etc/algernon/serverconf.lua'
        'usr/lib/systemd/system/algernon.service')
source=("git+https://github.com/xyproto/algernon#tag=$pkgver")
md5sums=('SKIP')
_gourl=github.com/xyproto/algernon

prepare() {
  export GOROOT=/usr/lib/go

  rm -rf build; mkdir -p build/go; cd build/go
  for f in "$GOROOT/"*; do ln -s "$f"; done
  rm pkg; mkdir pkg; cd pkg
  for f in "$GOROOT/pkg/"*; do ln -s "$f"; done

  export GOPATH="$srcdir/build"
  export GOROOT="$GOPATH/go"
  export DESTPATH="$GOPATH/src/$_gourl"

  # Make sure $DESTPATH is empty, but exists
  rm -rf "$DESTPATH"; mkdir -p "$DESTPATH"

  mv "$srcdir/$pkgname" "$(dirname $DESTPATH)"

  cd "$GOPATH/src/$_gourl"

  # Manage Go dependencies with Glide
  glide update
  glide install

  # Startup parameters
  setconf system/algernon_dev.service \
    ExecStart \
    "/usr/bin/algernon -e -a --domain --server --log /var/log/algernon.log --addr=:80 /srv/http"
  setconf system/algernon_dev.service Group=http
}

build() {
  cd "$GOPATH/src/$_gourl"

  go build -x
}

check() {
  cd "$GOPATH/src/$_gourl"

  go test
}

package() {
  cd "$GOPATH/src/$_gourl"

  install -Dm755 algernon "$pkgdir/usr/bin/algernon"
  install -Dm755 desktop/mdview "$pkgdir/usr/bin/mdview"
  install -Dm644 system/algernon_dev.service "$pkgdir/usr/lib/systemd/system/algernon.service"
  install -Dm644 system/logrotate "$pkgdir/etc/logrotate.d/algernon"
  install -Dm644 system/serverconf.lua "$pkgdir/etc/algernon/serverconf.lua"
  install -Dm644 desktop/algernon.desktop "$pkgdir/usr/share/applications/algernon.desktop"
  install -Dm644 desktop/markdown.png "$pkgdir/usr/share/pixmaps/markdown.png"
  install -d "$pkgdir/usr/share/doc/$pkgname/"
  cp -r samples "$pkgdir/usr/share/doc/algernon/samples"
  install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}

# vim: ts=2 sw=2 et:
