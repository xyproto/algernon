#!/bin/sh
go build && ./algernon --server --nolimit --addr=:3001 issue7/server.lua
