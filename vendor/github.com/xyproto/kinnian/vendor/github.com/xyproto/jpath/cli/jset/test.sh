#!/bin/sh
cp books.orig books.json
go run main.go books.json x[1].author Suzanne
diff -y books.orig books.json
