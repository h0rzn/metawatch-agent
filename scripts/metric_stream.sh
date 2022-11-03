#!/bin/sh

CONTAINER=$1

websocat --exit-on-eof ws://localhost:8080/containers/$CONTAINER/stream