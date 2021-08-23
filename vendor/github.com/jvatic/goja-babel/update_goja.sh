#!/bin/bash

set -euo pipefail

GOJA_REPO="github.com/dop251/goja"
CURRENT_VERSION=$(go mod edit --json | jq -r ".Require[] | select(.Path == \"$GOJA_REPO\") | .Version | .")
LATEST_VERSION=$(go list --json -u -m $GOJA_REPO | jq -r ".Update.Version")

if [ "$LATEST_VERSION" == "null" ]
then
    LATEST_VERSION=$CURRENT_VERSION
fi

echo "CURRENT_VERSION=$CURRENT_VERSION" >> $GITHUB_ENV
echo "LATEST_VERSION=$LATEST_VERSION" >> $GITHUB_ENV

echo "CURRENT_VERSION=$CURRENT_VERSION"
echo "LATEST_VERSION=$LATEST_VERSION"

if [ "$LATEST_VERSION" == "$CURRENT_VERSION" ]
then
    echo "Already at latest version: $CURRENT_VERSION"
    echo "UPDATE=NOOP" >> $GITHUB_ENV
else
    echo "Updating to latest version: $CURRENT_VERSION -> $LATEST_VERSION"
    go get -u github.com/dop251/goja@$LATEST_VERSION
    echo "UPDATE=$LATEST_VERSION" >> $GITHUB_ENV
fi
