#!/bin/sh
cd ..
docker build --no-cache -t algernon_lua_interpreter -f docker/lua_interpreter/Dockerfile .
