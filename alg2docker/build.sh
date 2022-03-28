#!/bin/sh
./alg2docker -f hello.alg Dockerfile
docker build -t hello .
