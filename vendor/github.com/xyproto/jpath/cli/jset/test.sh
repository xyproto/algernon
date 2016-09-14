#!/bin/sh
cp books.orig books.json
go run main.go books.json x[1].author Catniss
diff -y books.orig books.json
