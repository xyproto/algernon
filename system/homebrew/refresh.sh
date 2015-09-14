#!/bin/sh
cd ../..
go get -d -t -u
go get github.com/samertm/homebrew-go-resources
homebrew-go-resources > system/homebrew/resources.txt
cd system/homebrew
python res2lines.py > new_hashes.txt
rm resources.txt
echo '"new_hashes.txt" is ready.'
echo 'Remember that golang.org modules are handled separately.'
