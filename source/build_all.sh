#!/bin/sh
# Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
set -eu

cd "$(dirname "$0")"
mkdir -p dist
for artifact in dist/citation-ledger-go-* dist/SHA256SUMS.txt; do
  if [ -e "$artifact" ]; then
    rm -f "$artifact"
  fi
done

version=$(sed -n 's/.*appVersion[[:space:]]*=[[:space:]]*"\([^"]*\)".*/\1/p' model.go)
if [ -z "$version" ]; then
  echo "could not determine appVersion" >&2
  exit 1
fi

build() {
  target_os=$1
  target_arch=$2
  suffix=$3
  echo "building ${target_os}/${target_arch}"
  CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" \
    go build -buildvcs=false -trimpath -ldflags="-s -w" -o "dist/citation-ledger-go-${version}-${target_os}-${target_arch}${suffix}" .
}

unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "Go source is not formatted:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go test -count=1 ./...
go vet ./...
build linux amd64 ""
build windows amd64 ".exe"
build darwin amd64 ""
build darwin arm64 ""

if command -v sha256sum >/dev/null 2>&1; then
  (cd dist && sha256sum citation-ledger-go-*) > dist/SHA256SUMS.txt
elif command -v shasum >/dev/null 2>&1; then
  (cd dist && shasum -a 256 citation-ledger-go-*) > dist/SHA256SUMS.txt
else
  echo "sha256sum or shasum is required to create release checksums" >&2
  exit 1
fi

echo "Built artifacts in dist/"
