#!/bin/sh

ID=c6784e1e135cf97e1386d13aaa03ed5cf04593e0da5cff6291191ffc2b1c0c21
URL=http:/localhost/containers/$ID/stats?stream=false

curl --unix-socket /var/run/docker.sock $URL | python3 -m json.tool