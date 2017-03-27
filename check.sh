#!/bin/sh

[ -z $GOPATH ] && export GOPATH=~/go

echo -n go fmt...
go fmt && echo ok || echo fail

echo -n go vet...
go vet && echo ok || echo fail

# github.com/golang/lint/golint
echo -n golint...
golint && echo ok || echo fail

# tools from https://github.com/dominikh/go-tools
echo -n gosimple...
gosimple && echo ok || echo fail

echo -n unused...
unused && echo ok || echo fail

echo -n staticcheck...
staticcheck && echo ok || echo fail
