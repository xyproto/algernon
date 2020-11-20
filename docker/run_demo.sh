#!/bin/sh
# The --publish argument first takes the local port and then the container port
# The -v argument first takes the local directory and then the container directory name
docker run -v `pwd`/serve:/srv/algernon -v `pwd`/config:/etc/algernon -i -t --rm --publish 4000:4000 algernon_demo
