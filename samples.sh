#!/bin/sh
echo 'Please visit http://localhost:3000/'
./algernon --conf serverconf.lua --dir samples --httponly --debug --bolt --server "$@"
