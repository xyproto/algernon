#!/bin/sh
go build && ./algernon --server --nolimit --addr=:3001 regtest/issue7/server.lua
