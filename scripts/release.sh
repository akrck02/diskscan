#!/usr/bin/env bash
# Builds the release archives into dist/: universal (Apple Silicon + Intel)
# binaries of diskscan-ui and diskscan for macOS 13 and later, zipped, with
# SHA-256 checksums.
#
#   scripts/release.sh v0.1.0
set -euo pipefail

version="${1:?usage: scripts/release.sh <version, e.g. v0.1.0>}"
minimum_macos="13.0"

cd "$(dirname "$0")/.."
rm -rf dist
mkdir -p dist/build

# Each architecture is built on its own (cgo needs a C compiler per target),
# then lipo joins them into one binary.
build() {
	local name="$1" package="$2" arch="$3" goarch="$4"
	CGO_ENABLED=1 GOOS=darwin GOARCH="$goarch" \
		CC="clang -arch $arch" \
		MACOSX_DEPLOYMENT_TARGET="$minimum_macos" \
		CGO_CFLAGS="-mmacosx-version-min=$minimum_macos" \
		CGO_LDFLAGS="-mmacosx-version-min=$minimum_macos" \
		go build -trimpath -ldflags="-s -w" -o "dist/build/$name-$arch" "$package"
}

for program in "diskscan-ui ./cmd/diskscan-ui" "diskscan ."; do
	read -r name package <<<"$program"
	build "$name" "$package" arm64 arm64
	build "$name" "$package" x86_64 amd64
	lipo -create -output "dist/build/$name" "dist/build/$name-arm64" "dist/build/$name-x86_64"
	codesign --force --sign - "dist/build/$name" # ad-hoc: Apple Silicon runs nothing unsigned
	(cd dist/build && zip -q -X "../$name-$version-macos-universal.zip" "$name") # -X: no ._ attribute files
done

(cd dist && shasum -a 256 ./*.zip | sed 's| \./| |' >checksums.txt)
rm -rf dist/build
ls -l dist
