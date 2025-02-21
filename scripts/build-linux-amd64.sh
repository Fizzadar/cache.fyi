#!/bin/sh

set -euxo pipefail

CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ GOARCH=amd64 GOOS=linux CGO_ENABLED=1 go build -ldflags "-linkmode external -extldflags -static" -o ./dist/cachefyi-linux-amd64 ./cmd/cachefyi

scp -P 1022 ./dist/cachefyi-linux-amd64 winsauce.fizzadar.com:
