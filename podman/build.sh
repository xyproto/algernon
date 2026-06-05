#!/bin/sh
# Usage: ./build.sh [dev|interactive|lua|prod]

BUILD_TYPE=$1
if [ -z "$BUILD_TYPE" ]; then
    echo 'Please specify a build type:'
    echo '  dev | interactive | lua | prod'
    exit 1
fi

cd ..
podman build --no-cache -t algernon_$BUILD_TYPE -f podman/$BUILD_TYPE/Containerfile .
