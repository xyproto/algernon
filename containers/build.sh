#!/bin/sh
# Usage: ./build.sh [dev|interactive|lua|prod]

BUILD_TYPE=$1
if [ -z "$BUILD_TYPE" ]; then
    echo 'Please specify a build type:'
    echo '  dev | interactive | lua | prod'
    exit 1
fi

if command -v podman >/dev/null 2>&1; then
    RUNTIME=podman
elif command -v docker >/dev/null 2>&1; then
    RUNTIME=docker
else
    echo 'Error: neither podman nor docker found in PATH'
    exit 1
fi

cd ..
$RUNTIME build --no-cache -t algernon_$BUILD_TYPE -f containers/$BUILD_TYPE/Containerfile .
