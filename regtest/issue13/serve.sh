#!/bin/sh
elementary_iso_file=~/download/elementaryos-5.0-stable.20181016.iso
if [ ! -f "$elementary_iso_file" ]; then
  echo 'Please modifiy serve.sh so that it serves a directory with a really large file'
  exit 1
fi
mkdir -p /tmp/pub
ln -sf "$elementary_iso_file" /tmp/pub/elementary.iso
go build && ./algernon -x  /tmp/pub
