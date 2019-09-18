#!/bin/sh
echo 'Try editing the markdown file in samples/welcome and see the'
echo 'results instantly in the browser at http://localhost:3000/'
./algernon --dev --conf serverconf.lua --dir samples/welcome --httponly --debug --autorefresh --bolt --luapath luamodules --server "$@"
