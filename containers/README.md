# Containers

Container images for running Algernon with Docker or Podman.

The `Containerfile` format works with both `docker build` and `podman build`. The helper scripts auto-detect which runtime is available.

## Quick start

    ./build.sh dev
    ./run.sh dev

Then visit `http://localhost:3000`.

## Build types

| Type          | Description                                  | Port(s)   |
|---------------|----------------------------------------------|-----------|
| `prod`        | HTTP on port 80, HTTPS+HTTP/2 on port 443    | 8080/8443 |
| `dev`         | Development mode with auto-refresh           | 3000      |
| `interactive` | Development mode with interactive Lua prompt | 4000      |
| `lua`         | Lua interpreter only, no server              | —         |

## Deploying with systemd (Quadlet)

Copy `algernon.container` to `~/.config/containers/systemd/` (rootless) or `/etc/containers/systemd/` (rootful), then:

    systemctl --user daemon-reload
    systemctl --user start algernon.service

## Deploying with podman kube play

    podman kube play algernon-pod.yaml

## alg2docker

The `alg2docker/` subdirectory contains a utility for generating a Dockerfile from an `.alg` application file. See `alg2docker/README.md`.
