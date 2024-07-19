#!/bin/sh
# Usage: ./build.sh [dev|interactive|lua|prod]

BUILD_TYPE=$1
if [ -z "$BUILD_TYPE" ]; then
    echo 'Please specify a build type:'
    echo '  dev | interactive | lua | prod'
    exit 1
fi

cd ..
docker build --platform linux/amd64 --no-cache -t algernon_$BUILD_TYPE -f docker/$BUILD_TYPE/Dockerfile .
