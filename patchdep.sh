#!/bin/sh
# To make GOOS=linux GOARCH=arm GOARM=7 also work:
sed -i 's/length  int$/length  int64/g' vendor/github.com/pingcap/tidb/pkg/parser/mysql/util.go
sed -i 's/(flen int,/(flen int64,/g' vendor/github.com/pingcap/tidb/pkg/parser/mysql/util.go
sed -i 's/type SQLMode int/type SQLMode int64/g' vendor/github.com/pingcap/tidb/pkg/parser/mysql/const.go
