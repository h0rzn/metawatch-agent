#!/bin/sh

CONTAINER=$1

websocat ws://localhost:8080/containers/$CONTAINER/stream