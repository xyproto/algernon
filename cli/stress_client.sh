#!/bin/sh
time for url in \
  'http://localhost:3000/' \
  'http://localhost:3000/TODO.md' \
  'http://localhost:3000/samples/amber/' \
  'http://localhost:3000/samples/lua/' \
  'http://localhost:3000/samples/greetings/' \
  'http://localhost:3000/samples/pongo2/' \
  ;
do
  echo "$url"
  ab -n 5000 -c 100 -s 15 -H 'Accept-Encoding: gzip' "$url"
done
