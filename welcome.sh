#!/bin/sh

min_ulimit=512
ulimit=$(ulimit -n 2> /dev/null || echo "not found")
kernmax=$(sysctl kern.maxfiles 2> /dev/null || echo "not found")

if [ "$kernmax" = "kern.maxfiles: 245760" ]; then
  true # probably Ventura
else
  if [ "$ulimit" = "not found" ]; then
    echo 'NOTE:'
    echo "  'ulimit -n' could not be run."
    echo "  'ulimit -n' returns the number of files that can be open at the same time."
    echo "  The '--autorefresh' flag will not work if the ulimit is too low."
    exit 1
  fi
  if [ "$ulimit" -lt "$min_ulimit" ]; then
    echo 'NOTE:'
    echo "  'ulimit -n' on your system returns $ulimit."
    echo "  'ulimit -n' returns the number of files that can be open at the same time."
    echo "  $ulimit is too low to be able to use the '--autorefresh' flag."
    echo "  Increase the ulimit on your system to at least $min_ulimit to be able to use it."
    exit 1
  fi
fi

echo 'Try editing the markdown file in "samples" directory and see the'
echo 'results instantly in the browser at http://localhost:3000/'

./algernon --dev --conf serverconf.lua --dir samples --httponly --debug --autorefresh --bolt --server "$@"
