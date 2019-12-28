#!/bin/bash

set -e

main() {
    curl -o babel.js https://unpkg.com/babel-standalone@6.26.0/babel.min.js

    local dir=$(pwd)
    local bindataGitDir=$GOPATH/src/github.com/jvatic/go-bindata
    if [[ -d $bindataGitDir ]]; then
        cd $bindataGitDir
        git pull --ff-only origin master
    else
        git clone https://github.com/jvatic/go-bindata/ $bindataGitDir
    fi
    cd $bindataGitDir/go-bindata
    go build ./
    cd $dir
    $bindataGitDir/go-bindata/go-bindata -nomemcopy -nocompress -pkg "babel" ./babel.js
    go test ./
}

main $@
