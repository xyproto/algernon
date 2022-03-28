#!/bin/sh

port8080=${1:-yes}
if [[ "$port8080" == "yes" ]]; then
  echo
  echo
  echo "It's possible to visit http://localhost:8080/ after the docker image has launched"
  echo
  echo
  docker run --mount type=bind,source=$PWD/config,destination=/etc/algernon,readonly --publish 8080:80 --rm hello
else
  echo
  echo
  echo "It's possible to visit http://localhost/ and https://localhost/ after the docker image has launched, if docker has the right permissions"
  echo
  echo
  docker run --mount type=bind,source=$PWD/config,destination=/etc/algernon,readonly --publish 80:80 --publish 443:443 --rm hello
fi
