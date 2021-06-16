#!/bin/bash

set -euo pipefail

LATEST_VERSION=$(curl -i https://unpkg.com/@babel/standalone | grep -i Location | grep -Eo '[.0-9]+')
CURRENT_VERSION=$(cat babel_version.txt)

echo "CURRENT_VERSION=$CURRENT_VERSION" >> $GITHUB_ENV
echo "LATEST_VERSION=$LATEST_VERSION" >> $GITHUB_ENV

if [ "$LATEST_VERSION" == "$CURRENT_VERSION" ]
then
    echo "Already at latest version: $CURRENT_VERSION"
    echo "UPDATE=NOOP" >> $GITHUB_ENV
else
    echo "Updating to latest version: $CURRENT_VERSION -> $LATEST_VERSION"
    curl -o babel.js https://unpkg.com/@babel/standalone@$LATEST_VERSION/babel.min.js
    echo $LATEST_VERSION > babel_version.txt
    echo "UPDATE=$LATEST_VERSION" >> $GITHUB_ENV
fi
