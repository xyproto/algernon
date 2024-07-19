#!/bin/sh

docker tag algernon_prod xyproto/algernon:prod
docker tag algernon_dev xyproto/algernon:dev
docker tag algernon_interactive xyproto/algernon:interactive
docker tag algernon_lua xyproto/algernon:lua
docker tag xyproto/algernon:interactive xyproto/algernon:latest

docker push xyproto/algernon:prod
docker push xyproto/algernon:dev
docker push xyproto/algernon:interactive
docker push xyproto/algernon:lua
docker push xyproto/algernon:latest
