#!/bin/sh
echo 'Please visit https://localhost:3000/'
./algernon --conf serverconf.lua --dir samples/bob --debug --bolt --server "$@"
