#!/bin/bash
echo -ne 'Testing HTTP server...\t'
./algernon --quiet --httponly --server --boltdb /tmp/_bolt_test.db --addr :45678 &
sleep 10
output=$(curl -sIm3 -o- http://localhost:45678)
if [[ $output == *"Server: Algernon"* ]]; then
  echo ok
else
  echo FAIL
  exit 1
fi
kill $(ps -ef | grep algernon | grep :45678 | cut -d" " -f4) 2>/dev/null
exit 0
