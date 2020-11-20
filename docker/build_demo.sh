#!/bin/sh
cd ..
docker build --no-cache -t algernon_demo -f docker/demo/Dockerfile .
