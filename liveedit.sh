#!/bin/sh
echo 'Try editing the markdown file in samples/greetings and see the'
echo 'results instantly in the browser at http://localhost:3000/'
./algernon --dev --conf serverconf.lua --dir samples/greetings --httponly --debug --autorefresh --bolt --server "$@"
