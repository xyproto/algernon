#!/bin/bash
echo -ne 'Launching Algernon...\t'
./algernon --quiet --httponly --server --nodb --addr :45678 &
PID=$!
function finish {
  echo -ne "Stopping PID $PID...\t"
  kill $PID 2>/dev/null && echo ok || echo fail
}
trap finish EXIT
echo ok
echo -ne 'Waiting for response...\t'
for i in $(seq 1 30); do curl -sIm3 -o/dev/null http://localhost:45678 && break || sleep 1; done
output=$(curl -sIm3 -o- http://localhost:45678)
if [[ $output == *"Server: Algernon"* ]]; then
  echo ok
else
  echo fail
  exit 1
fi
