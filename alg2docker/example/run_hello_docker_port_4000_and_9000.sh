#!/bin/sh
# The --publish argument first takes the local port and then the container port
# The -v argument first takes the local directory and then the container directory name
echo 'Serving HTTP on port 4000 and HTTPS or HTTP/2 on port 9000'
docker run -v `pwd`/serve:/srv/algernon -v `pwd`/config:/etc/algernon --rm --publish 4000:80 --publish 9000:443 hello
