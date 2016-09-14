#!/bin/sh
cp abc.orig abc.json
go run main.go abc.json b
diff -y abc.orig abc.json
