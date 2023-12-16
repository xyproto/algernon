#!/bin/sh
# Usage: ./run.sh [dev|interactive|lua|prod]

RUN_TYPE=$1
if [ -z "$RUN_TYPE" ]; then
    echo "Please specify a run type (dev, interactive, lua, prod)"
    exit 1
fi

VOLUME_ARGS="-v `pwd`/serve:/srv/algernon -v `pwd`/config:/etc/algernon"
PUBLISH_ARGS="--publish 3000:3000"  # default to dev settings

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
    PUBLISH_ARGS="--publish 80:80 --publish 443:443"
    ;;
  *)
    echo "Invalid run type: $RUN_TYPE"
    exit 1
    ;;
esac

docker run $VOLUME_ARGS --rm $PUBLISH_ARGS algernon_$RUN_TYPE
