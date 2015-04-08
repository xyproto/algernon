#!/bin/sh
echo 'Please visit https://localhost:3000/'
./algernon -conf server.lua -dir examples/bob "$@"
#./algernon -httponly -conf server.lua -dir examples/bob "$@"
