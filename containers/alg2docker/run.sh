#!/bin/bash

# Use available command to determine if port is open
is_port_open() {
  local port=$1

  if command -v netstat >/dev/null 2>&1; then
    netstat -tuln | grep -q ":$port "
    return $?
  elif command -v lsof >/dev/null 2>&1; then
    lsof -i :$port | grep -q LISTEN
    return $?
  elif command -v sockstat >/dev/null 2>&1; then
    sockstat -4l | grep -q ":$port "
    return $?
  else
    echo "Error: Could not find netstat, lsof, or sockstat. Unable to check if port is open."
    exit 1
  fi
}

port8080=${1:-yes}

if [[ "$port8080" == "yes" ]]; then
  if is_port_open 8080; then
    echo "Error: Port 8080 is already in use!"
    exit 1
  fi
  echo
  echo "It's possible to visit http://localhost:8080/ after the docker image has launched"
  echo
  docker run --mount type=bind,source="$PWD/config",destination=/etc/algernon,readonly --publish 8080:80 --rm hello
else
  if is_port_open 80 || is_port_open 443; then
    echo "Error: Either Port 80 or 443 (or both) are already in use!"
    exit 1
  fi
  echo
  echo "It's possible to visit http://localhost/ and https://localhost/ after the docker image has launched, if docker has the right permissions"
  echo
  docker run --mount type=bind,source="$PWD/config",destination=/etc/algernon,readonly --publish 80:80 --publish 443:443 --rm hello
fi
