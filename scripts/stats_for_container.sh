#!/bin/sh

URL=http:/localhost/containers/$1/stats?stream=false

curl --unix-socket /var/run/docker.sock $URL | python3 -m json.tool