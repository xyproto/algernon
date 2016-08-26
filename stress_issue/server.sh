#!/bin/sh
go build -race && ./algernon -t -c --cachesize=10000000 --nolimit . :9000
