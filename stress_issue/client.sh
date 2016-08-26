#!/bin/sh
time for url in \
  'http://localhost:9000/' \
  'http://localhost:9000/TODO.md' \
  'http://localhost:9000/samples/amber/' \
  'http://localhost:9000/samples/hellolua/' \
  'http://localhost:9000/samples/greetings/' \
  'http://localhost:9000/samples/pongo2/' \
  ;
do
  echo "$url"
  ab -n 5000 -c 100 -s 15 -H 'Accept-Encoding: gzip' "$url"
done
