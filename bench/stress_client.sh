#!/bin/sh
time for url in \
  'http://localhost:7531/' \
  'http://localhost:7531/TODO.md' \
  'http://localhost:7531/samples/sass/' \
  'http://localhost:7531/samples/hellolua/' \
  'http://localhost:7531/samples/greetings/' \
  'http://localhost:7531/samples/pongo2/' \
  ;
do
  echo "$url"
  ab -n 5000 -c 100 -s 15 -H 'Accept-Encoding: gzip' "$url"
done
