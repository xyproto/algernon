#!/bin/sh
cd "$(dirname $(readlink -f $0))"
truncate -s 10T large-file
(cd ../..; go build && ./algernon -x regtest/issue13)
