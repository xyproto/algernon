#!/bin/sh

# Build
CGO_ENABLED=0 GOOS=linux go build -v -trimpath -ldflags "-s" -a

# Compress
upx algernon || echo 'Not using upx'

# Package
VERSION="$(grep '* Version:' README.md | cut -d' ' -f3)"
mkdir -p "algernon-$VERSION"
mv -v algernon "algernon-$VERSION"
tar zcvf "algernon-$VERSION-static_linux.tar.gz" "algernon-$VERSION"
rm -rf "algernon-$VERSION"

# Size
du -h "algernon-$VERSION-static_linux.tar.gz"
