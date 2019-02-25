#!/bin/sh
./algernon --dev --conf serverconf.lua --dir ./samples/welcome --httponly --debug --autorefresh --bolt --server -n &
PID=$!
if ps -p $PID > /dev/null; then
  echo "$PID is running"
else
  echo 'Algernon could not start. Try launching Algernon manually.'
  exit 1
fi
echo 'Attacking Algernon for 30 seconds on port 3000, in a way that uses the Lua engine'
echo "GET http://127.0.0.1:3000/counter/" | vegeta attack -rate=500 -duration=30s | tee /tmp/results.bin | vegeta report
function finish {
  echo -ne "Stopping PID $PID...\t"
  kill $PID 2>/dev/null && echo ok || echo fail
}
trap finish EXIT
