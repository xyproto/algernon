#!/bin/sh
app=hello
../alg2docker "$@" "$app.alg" && docker build -t "$app" .
