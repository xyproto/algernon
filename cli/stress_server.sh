#!/bin/sh
# The "-race" flag gets in the way of the CPU profiling
go build && ./algernon -t -c --cachesize=10000000 --nolimit --cpuprofile=algernon.prof
#echo 'Now run: "go tool pprof -alloc_objects algernon algernon.prof" and type "weblist"'
echo 'Now run: "go tool pprof algernon algernon.prof" and type "weblist"'
