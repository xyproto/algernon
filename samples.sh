#!/bin/sh
echo 'Please visit http://localhost:3000/'
./algernon -conf server.lua -dir samples "$@" -httponly -autorefresh
