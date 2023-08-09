#!/bin/sh

go build -mod=vendor -tags=trace && ./algernon -n -t -c --cachesize=10000000 --nolimit --cpuprofile=algernon.prof . :7531

# The "-race" flag gets in the way of the CPU profiling, unfortunately
#go build -mod=vendor -race && ./algernon -n -t -c --cachesize=10000000 --nolimit . :7531

echo 'Now run: "go tool pprof algernon algernon.prof" and try "top50"'
