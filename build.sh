#!/bin/bash
set -e

BINARY="dtchat"
BUILD_DIR="releases"

PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    #"linux/amd64"
    #"linux/arm64"
    "windows/amd64"
)

echo "=== dtchat 多平台构建 ==="
rm -rf "$BUILD_DIR" && mkdir -p "$BUILD_DIR"

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r os arch <<< "$platform"
    output="${BUILD_DIR}/${BINARY}-${os}-${arch}"
    [ "$os" = "windows" ] && output="${output}.exe"
    echo "构建 ${os}/${arch}..."
    GOOS=$os GOARCH=$arch go build -ldflags "-s -w" -trimpath -o "$output" .
    echo "  -> $output ($(du -h "$output" | cut -f1))"
done

echo ""
echo "构建完成，产物在 ${BUILD_DIR}/"
