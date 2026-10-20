#!/usr/bin/env bash
# build.sh — 交叉编译 campus-net-helper 的全部平台产物并打包
#
# 用法：
#   ./build.sh                 # 版本号取自最近的 git tag，无 tag 时使用 dev
#   ./build.sh v0.1.0          # 显式指定版本号（正式发版推荐）
#
# 产物（位于 dist/）：
#   campus-net-helper_<版本>_{linux-amd64,linux-arm64,darwin-amd64,darwin-arm64}.tar.gz
#   campus-net-helper_<版本>_windows-amd64.zip
#   sha256sums.txt
#
# 依赖：go、tar、zip

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ---------------------------------------------------------------
# 版本号
# ---------------------------------------------------------------
if [[ $# -ge 1 && -n "$1" ]]; then
  RAW_VERSION="$1"
else
  RAW_VERSION="$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo dev)"
fi

# 去掉可能携带的 v 前缀，保证产物名里只有一个版本号
VERSION="${RAW_VERSION#v}"

# 去掉预发布标识后的基础版本（仅用于显示）
BASE_VERSION="${VERSION%%-*}"

echo "==> 构建版本号 : $VERSION"
echo "==> 提交       : $(git rev-parse --short HEAD 2>/dev/null || echo unknown)"

# ---------------------------------------------------------------
# 依赖检查
# ---------------------------------------------------------------
require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "错误：缺少命令 '$1'，请先安装。" >&2
    exit 1
  }
}
require go
require tar
require zip

# ---------------------------------------------------------------
# 公共构建参数
# ---------------------------------------------------------------
DIST="$SCRIPT_DIR/dist"
BUILD="$DIST/build"
PKG_DIR() { # $1 = target
  echo "$BUILD/$1"
}

LDFLAGS="-s -w \
  -X main.version=$VERSION \
  -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
  -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# 需要包含在压缩包里的额外文件（存在才打包）
EXTRA_FILES=()
[[ -f LICENSE    ]] && EXTRA_FILES+=(LICENSE)
[[ -f README.md  ]] && EXTRA_FILES+=(README.md)
[[ -f CHANGELOG.md ]] && EXTRA_FILES+=(CHANGELOG.md)

# ---------------------------------------------------------------
# 目标平台列表：GOOS GOARCH EXT
# ---------------------------------------------------------------
TARGETS=(
  "linux  amd64  ""
  "linux  arm64  ""
  "darwin amd64  ""
  "darwin arm64  ""
  "windows amd64 .exe"
)

rm -rf "$DIST"
mkdir -p "$DIST"

# ---------------------------------------------------------------
# 逐个目标构建 + 打包
# ---------------------------------------------------------------
for spec in "${TARGETS[@]}"; do
  # 以空格分列，最后一个是扩展名（可能为空）
  read -r GOOS GOARCH EXT <<< "$spec"
  GOOS="${GOOS:-linux}"
  GOARCH="${GOARCH:-amd64}"
  EXT="${EXT:-}"

  TARGET="${GOOS}-${GOARCH}"
  BINARY="campus-net-helper_${VERSION}_${TARGET}${EXT}"

  echo "==> [$TARGET] 交叉编译中..."
  STAGEDIR=$(PKG_DIR "$TARGET")
  mkdir -p "$STAGEDIR"

  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags "$LDFLAGS" \
      -o "$STAGEDIR/$BINARY" ./main.go

  # 放入随附文件
  for f in "${EXTRA_FILES[@]}"; do
    cp "$SCRIPT_DIR/$f" "$STAGEDIR/"
  done
done

# ---------------------------------------------------------------
# 打包为 tar.gz / zip
# ---------------------------------------------------------------
echo "==> 打包产物..."
cd "$BUILD"

pack_targets=(
  "linux-amd64:tar"
  "linux-arm64:tar"
  "darwin-amd64:tar"
  "darwin-arm64:tar"
  "windows-amd64:zip"
)

for entry in "${pack_targets[@]}"; do
  TARGET="${entry%%:*}"
  KIND="${entry##*:}"
  DIRNAME="campus-net-helper_${VERSION}_${TARGET}"

  if [[ "$KIND" == "tar" ]]; then
    # tar：压缩包顶层为目录 campus-net-helper_版本_目标/
    OUT="${DIST}/campus-net-helper_${VERSION}_${TARGET}.tar.gz"
    rm -f "$OUT"
    tar -C "$BUILD" -czf "$OUT" "$DIRNAME"
  else
    # zip：顶层含目录 campus-net-helper_版本_windows-amd64/（Windows 解压体验更好）
    OUT="${DIST}/campus-net-helper_${VERSION}_${TARGET}.zip"
    rm -f "$OUT"
    ( cd "$BUILD" && zip -rX "$OUT" "$DIRNAME" )
  fi
done

# ---------------------------------------------------------------
# 生成 SHA-256 校验和
# ---------------------------------------------------------------
cd "$DIST"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum *.tar.gz *.zip > sha256sums.txt
else
  # macOS / BSD
  shasum -a 256 *.tar.gz *.zip > sha256sums.txt
fi

echo "==> 完成。产物清单："
ls -1 "$DIST"/*.tar.gz "$DIST"/*.zip "$DIST"/sha256sums.txt

# 打印校验命令提示
echo
echo "本地校验：cd $DIST && sha256sum -c sha256sums.txt"
