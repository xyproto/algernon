#!/bin/sh
time for url in \
  'http://localhost:3000/' \
  'http://localhost:3000/TODO.md' \
  'http://localhost:3000/samples/amber/' \
  'http://localhost:3000/samples/hellolua/' \
  'http://localhost:3000/samples/greetings/' \
  ;
do
  echo "$url"
  ab -n 5000 -c 100 -s 15 -H 'Accept-Encoding: gzip' "$url"
done
