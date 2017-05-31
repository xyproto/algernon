#!/bin/sh
cp books.orig books.json
go run main.go books.json x '{"author": "Joan Grass", "book": "The joys of gardening"}'
diff -y books.orig books.json
