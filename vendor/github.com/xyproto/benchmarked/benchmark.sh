#!/bin/sh
go test -bench=. "$@" | tee bench.out
head -4 bench.out
grep 'ns/op' bench.out | sort -r -n -k3
