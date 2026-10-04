#!/usr/bin/env bash

set -euo pipefail

for arch in amd64 arm64; do
  make docker-images GOARCH="$arch"
  actual=$(docker image inspect prometheus-plex-exporter:latest --format '{{.Os}}/{{.Architecture}}')
  expected="linux/$arch"
  if [[ "$actual" != "$expected" ]]; then
    echo "expected $expected image, got $actual" >&2
    exit 1
  fi
done
