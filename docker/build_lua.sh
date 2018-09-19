#!/bin/sh
cd ..
docker build --no-cache -t algernon_lua -f docker/lua/Dockerfile .
