#!/bin/sh

set -euxo pipefail

gow -e yaml,go,html,css,js,sql run ./cmd/cachefyi -prettyLogs -debug $@
