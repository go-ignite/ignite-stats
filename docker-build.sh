#! /bin/bash

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ignite-stats
docker build -t $1 .
