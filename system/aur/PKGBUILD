# Maintainer: Alexander F. Rødseth <xyproto@archlinux.org>

pkgname=algernon
pkgver=1.12.4
pkgrel=1
pkgdesc='Small self-contained web server with Lua, Markdown, QUIC, Redis and PostgreSQL support'
arch=(x86_64)
url='https://algernon.roboticoverlords.org/'
license=(MIT)
makedepends=(git go)
optdepends=('mariadb: For using the MariaDB/MySQL database backend'
            'postgresql: For using the PostgreSQL database backend'
            'redis: For using the Redis database backend')
source=("git+https://github.com/xyproto/algernon#tag=$pkgver")
md5sums=('SKIP')

prepare() {
  cd "$pkgname"
  go build -gcflags "all=-trimpath=${PWD}" -asmflags "all=-trimpath=${PWD}" -ldflags "-extldflags ${LDFLAGS}"
}

package() {
  cd "$pkgname"

  install -Dm755 algernon "$pkgdir/usr/bin/algernon"
  install -Dm755 desktop/mdview "$pkgdir/usr/bin/mdview"
  install -Dm644 system/logrotate "$pkgdir/etc/logrotate.d/algernon"
  install -Dm644 system/serverconf.lua "$pkgdir/etc/algernon/serverconf.lua"
  install -Dm644 desktop/algernon.desktop \
    "$pkgdir/usr/share/applications/algernon.desktop"
  install -Dm644 desktop/algernon_md.desktop \
    "$pkgdir/usr/share/applications/algernon_md.desktop"
  install -Dm644 desktop/markdown.png "$pkgdir/usr/share/pixmaps/markdown.png"
  install -Dm644 system/algernon_dev.service \
    "$pkgdir/usr/share/doc/$pkgname/algernon.service.example"
  cp -r samples "$pkgdir/usr/share/doc/$pkgname/samples"
  install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}

# vim: ts=2 sw=2 et:
