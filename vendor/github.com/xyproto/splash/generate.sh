#!/bin/sh
# Generate cmd/selfupdate/styles.go based on the latest commit in the master branch on github.com/alcthomas/chroma
go run cmd/selfupdate/*.go
# Generate the HTML files for the Style Gallery in docs/
go run cmd/gendoc/*.go
# Serve docs/ with Algernon, and open the web page in a browser
algernon -n -o -t docs/
