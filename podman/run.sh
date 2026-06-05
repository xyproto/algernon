#!/bin/sh
# Usage: ./run.sh [dev|interactive|lua|prod]
#
# The ":Z" suffix on volume mounts tells podman to relabel the content with an
# SELinux MCS label private to this container. Drop it (or use ":z" for shared)
# on systems without SELinux. Without it, the container cannot read the volume
# on Fedora / RHEL / CentOS Stream.

RUN_TYPE=$1
if [ -z "$RUN_TYPE" ]; then
    echo 'Please specify a run type:'
    echo '  dev | interactive | lua | prod'
    exit 1
fi

VOLUME_ARGS="-v `pwd`/serve:/srv/algernon:Z -v `pwd`/config:/etc/algernon:Z"
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
    # Rootless podman cannot bind privileged ports (<1024) by default. Map to
    # 8080/8443 instead, or run as root / set net.ipv4.ip_unprivileged_port_start=80
    # to use 80/443 directly.
    PUBLISH_ARGS="--publish 8080:80 --publish 8443:443"
    ;;
  *)
    echo "Invalid run type: $RUN_TYPE"
    exit 1
    ;;
esac

podman run $VOLUME_ARGS --rm $PUBLISH_ARGS algernon_$RUN_TYPE
