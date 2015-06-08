#!/bin/sh
cd ..
zip -r withplugins.zip withplugins
cd ..
if [[ $1 == "b" ]]; then
  go build -v -race && go fmt *.go || exit 1
fi
./algernon -h --verbose apps/withplugins.zip
