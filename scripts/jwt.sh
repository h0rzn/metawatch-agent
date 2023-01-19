#!/bin/sh
token=$(http --json POST localhost:8080/login username=master password=master | jq -r  '.token')

http -f GET localhost:8080/api/containers/all "Authorization:Bearer $token"  "Content-Type: application/json"