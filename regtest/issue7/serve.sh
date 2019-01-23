#!/bin/sh
cd "$(dirname $(readlink -f $0))"
(cd ../..; go build && ./algernon --server --nolimit --addr=:3001 regtest/issue7/server.lua)
