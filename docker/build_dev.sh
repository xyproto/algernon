#!/bin/sh
cd ..
docker build --no-cache -t algernon_dev -f docker/dev/Dockerfile .
