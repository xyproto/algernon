#!/bin/sh
echo 'Please visit http://localhost:3000/'
./algernon --conf serverconf.lua --dir samples/greetings --httponly --debug --autorefresh --bolt --server "$@"
