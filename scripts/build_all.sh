#!/bin/bash
set -e

echo "===================================================="
echo "  一键跨平台交叉编译: 阿里云 OSS 双向目录同步工具"
echo "===================================================="

mkdir -p release

export CGO_ENABLED=0

# 1. Windows 64-bit (.exe)
echo "📦 正在编译 Windows amd64..."
GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o release/aliyun-oss-sync.exe .

# 2. macOS Apple Silicon (M1/M2/M3/M4 arm64)
echo "📦 正在编译 macOS Apple Silicon (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o release/aliyun-oss-sync-darwin-arm64 .

# 3. macOS Intel (x86_64)
echo "📦 正在编译 macOS Intel (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o release/aliyun-oss-sync-darwin-amd64 .

# 4. Linux 64-bit (x86_64)
echo "📦 正在编译 Linux amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o release/aliyun-oss-sync-linux-amd64 .

# 5. Linux ARM64
echo "📦 正在编译 Linux arm64..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o release/aliyun-oss-sync-linux-arm64 .

echo "===================================================="
echo "✅ 全部平台编译成功！生成文件位于 release/ 目录下:"
ls -lh release/
