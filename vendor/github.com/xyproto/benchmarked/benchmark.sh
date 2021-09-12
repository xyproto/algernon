#!/bin/sh
go version | tee bench.out
go test -bench=. "$@" | tee -a bench.out
head -5 bench.out > sorted.out
grep 'ns/op' bench.out | sort -r -n -k3 >> sorted.out
tail -2 bench.out >> sorted.out
cat sorted.out
