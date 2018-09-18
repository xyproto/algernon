#!/bin/sh
cd ..
docker build --no-cache -t algernon_interactive -f docker/interactive/Dockerfile .
