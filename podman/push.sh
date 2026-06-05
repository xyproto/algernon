#!/bin/sh
# Push Algernon images to both docker.io and quay.io.
# Log in first: `podman login docker.io` and `podman login quay.io`.

for registry in docker.io quay.io; do
    podman tag algernon_prod        $registry/xyproto/algernon:prod
    podman tag algernon_dev         $registry/xyproto/algernon:dev
    podman tag algernon_interactive $registry/xyproto/algernon:interactive
    podman tag algernon_lua         $registry/xyproto/algernon:lua
    podman tag $registry/xyproto/algernon:interactive $registry/xyproto/algernon:latest

    podman push $registry/xyproto/algernon:prod
    podman push $registry/xyproto/algernon:dev
    podman push $registry/xyproto/algernon:interactive
    podman push $registry/xyproto/algernon:lua
    podman push $registry/xyproto/algernon:latest
done
