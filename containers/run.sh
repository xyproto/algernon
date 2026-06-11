#!/bin/sh
# Usage: ./run.sh [dev|interactive|lua|prod]

RUN_TYPE=$1
if [ -z "$RUN_TYPE" ]; then
    echo 'Please specify a run type:'
    echo '  dev | interactive | lua | prod'
    exit 1
fi

if command -v podman >/dev/null 2>&1; then
    RUNTIME=podman
elif command -v docker >/dev/null 2>&1; then
    RUNTIME=docker
else
    echo 'Error: neither podman nor docker found in PATH'
    exit 1
fi

VOLUME_ARGS="-v $(pwd)/serve:/srv/algernon -v $(pwd)/config:/etc/algernon"
PUBLISH_ARGS="--publish 3000:3000"

case $RUN_TYPE in
  "dev")
    PUBLISH_ARGS="--publish 3000:3000"
    ;;
  "interactive")
    PUBLISH_ARGS="-i -t --publish 4000:4000"
    ;;
  "lua")
    VOLUME_ARGS=""
    PUBLISH_ARGS="-i -t"
    ;;
  "prod")
    PUBLISH_ARGS="--publish 8080:80 --publish 8443:443"
    ;;
  *)
    echo "Invalid run type: $RUN_TYPE"
    exit 1
    ;;
esac

$RUNTIME run $VOLUME_ARGS --rm $PUBLISH_ARGS algernon_$RUN_TYPE
