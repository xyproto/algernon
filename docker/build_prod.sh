#!/bin/sh
cd ..
docker build --no-cache -t algernon_prod -f docker/prod/Dockerfile .
