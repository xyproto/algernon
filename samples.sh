#!/bin/sh
echo 'Please visit http://localhost:3000/'
./algernon -conf server.lua -dir examples "$@" -httponly -autorefresh
