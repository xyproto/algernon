#!/bin/sh
echo "Generating trace.out"
echo "You can visit http://localhost:3000/ in a couple of seconds"
./algernon --dev --conf serverconf.lua --dir samples --httponly --debug --bolt --server -tracefile trace.out "$@"
go tool trace trace.out
