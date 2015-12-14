#!/bin/sh
# The --publish argument first takes the local port and then the container port
# The -v argument first takes the local directory and then the container directory name
docker run -v `pwd`/serve:/srv/algernon -v `pwd`/config:/etc/algernon --rm --publish 3000:3000 algernon_dev
