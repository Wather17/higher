#!/usr/bin/env bash
set -euo pipefail

VERSION=${1:-}
if [[ -z "$VERSION" || ! "$VERSION" =~ ^[[:alnum:]][[:alnum:].+-]*$ ]]; then
	echo "Usage: $0 <version> (for example: dev or 1.2.3)" >&2
	exit 2
fi

COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-X github.com/Wather17/higher/internal/cli.Version=${VERSION} -X github.com/Wather17/higher/internal/cli.Commit=${COMMIT} -X github.com/Wather17/higher/internal/cli.BuildDate=${BUILD_DATE} -s -w"

mkdir -p dist

echo "Building Higher ${VERSION} for Linux amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o dist/higher-linux-amd64 ./cmd/higher

echo "Building Higher ${VERSION} for Linux arm64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$LDFLAGS" -o dist/higher-linux-arm64 ./cmd/higher

echo "Building Higher ${VERSION} for Windows amd64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o dist/higher-windows-amd64.exe ./cmd/higher

echo "Builds are available in ./dist/"
