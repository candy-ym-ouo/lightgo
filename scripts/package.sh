#!/bin/sh
set -eu
APP=lightgo-server
OS=$(go env GOOS)
ARCH=$(go env GOARCH)
NAME="lightgo_${OS}_${ARCH}"
rm -rf "dist/$NAME"
mkdir -p "dist/$NAME"
go build -trimpath -ldflags='-s -w' -o "dist/$NAME/$APP" ./cmd/server
cp -R web "dist/$NAME/web"
cp README.md "dist/$NAME/README.md"
LC_ALL=C tar -C dist -czf "dist/$NAME.tar.gz" "$NAME"
printf 'created %s\n' "dist/$NAME.tar.gz"
