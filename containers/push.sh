#!/bin/sh
# Push Algernon images to both docker.io and quay.io.

if command -v podman >/dev/null 2>&1; then
    RUNTIME=podman
elif command -v docker >/dev/null 2>&1; then
    RUNTIME=docker
else
    echo 'Error: neither podman nor docker found in PATH'
    exit 1
fi

for registry in docker.io quay.io; do
    $RUNTIME tag algernon_prod        $registry/xyproto/algernon:prod
    $RUNTIME tag algernon_dev         $registry/xyproto/algernon:dev
    $RUNTIME tag algernon_interactive $registry/xyproto/algernon:interactive
    $RUNTIME tag algernon_lua         $registry/xyproto/algernon:lua
    $RUNTIME tag $registry/xyproto/algernon:interactive $registry/xyproto/algernon:latest

    $RUNTIME push $registry/xyproto/algernon:prod
    $RUNTIME push $registry/xyproto/algernon:dev
    $RUNTIME push $registry/xyproto/algernon:interactive
    $RUNTIME push $registry/xyproto/algernon:lua
    $RUNTIME push $registry/xyproto/algernon:latest
done
