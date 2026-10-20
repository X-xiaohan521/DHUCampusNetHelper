#!/usr/bin/env bash
# 交叉编译全部平台二进制到 dist/，并生成 sha256sums.txt
#
#   ./build.sh            # 版本号取最近的 git tag（去掉 v 前缀），无 tag 时用 dev
#   ./build.sh v0.1.0     # 显式指定版本号（正式发版推荐）
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

VERSION="${1:-$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo dev)}"
VERSION="${VERSION#v}"
command -v go >/dev/null 2>&1 || { echo "错误：未找到 go，请先安装 Go 1.27+。" >&2; exit 1; }

LDFLAGS="-s -w \
  -X main.version=${VERSION} \
  -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
  -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

rm -rf dist && mkdir dist

# 依次构建：GOOS GOARCH
for t in "linux amd64" "linux arm64" "darwin amd64" "darwin arm64" "windows amd64"; do
  read -r goos goarch <<< "$t"
  name="campus-net-helper_${VERSION}_${goos}-${goarch}"
  [[ "$goos" == "windows" ]] && name="${name}.exe"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "dist/$name" .
  echo "已构建 dist/$name"
done

# 生成 SHA-256 校验和
cd dist
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum campus-net-helper_* > sha256sums.txt
else
  shasum -a 256 campus-net-helper_* > sha256sums.txt
fi

echo
ls -1
echo
echo "本地校验：sha256sum -c sha256sums.txt   # macOS 使用 shasum -c sha256sums.txt"
