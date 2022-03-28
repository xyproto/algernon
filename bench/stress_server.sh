#!/bin/sh
# The "-race" flag gets in the way of the CPU profiling
go build -mod=vendor && ./algernon -t -c --cachesize=10000000 --nolimit --cpuprofile=algernon.prof . :7531
#go build -race && ./algernon -t -c --cachesize=10000000 --nolimit . :7531
echo 'Now run: "go tool pprof algernon algernon.prof" and type "weblist"'
