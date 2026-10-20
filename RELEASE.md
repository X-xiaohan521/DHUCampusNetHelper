# 发布流程 (Release Guide)

本文件约定 `campus-net-helper` 的版本规范、二进制命名规范、构建流程与 GitHub Release Notes 模板，适用于手动构建与自动化构建两种场景。

## 1. 版本规范

版本号遵循 [语义化版本 2.0.0](https://semver.org/lang/zh-CN/)：

```
MAJOR.MINOR.PATCH[-PRE]
  │     │      │    └─ 预发布标识（可选），如 0.2.0-alpha.1
  │     │      └────── 向下兼容的缺陷修复
  │     └───────────── 向下兼容的新功能
  └─────────────────── 不兼容的变更（如环境变量改名、认证流程变更）
```

项目当前处于 `0.x` 阶段，环境变量与认证流程约定仍可能调整；达到稳定后以 `1.0.0` 作为首个正式稳定版。

发版时通过 git 标签标记版本：

```sh
git tag -s v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

推送 tag 即触发自动化构建（见 3.3），二进制中内嵌的版本信息必须与该 tag 一致。

`main.go` 需支持 `--version`，打印由构建时注入的版本信息：

```go
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func init() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("campus-net-helper %s (commit %s, built %s)\n", version, commit, buildTime)
		os.Exit(0)
	}
}
```

构建时通过 `ldflags` 注入：

```sh
CGO_ENABLED=0 go build \
  -ldflags "-s -w \
    -X main.version=${TAG#v} \
    -X main.commit=$(git rev-parse --short HEAD) \
    -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o dist/build/<target>/campus-net-helper ./main.go
```

## 2. 二进制命名规范

遵循 `程序名_版本_操作系统-架构` 的形式，与主流 Go 项目（如 goreleaser 默认产物）保持一致：

```
campus-net-helper_<版本号>_<GOOS>-<GOARCH>[.exe]
```

各平台的正式发布产物如下（以版本 `0.1.0` 为例）：

| 平台    | 架构              | 压缩包内文件名                              | 发布资产（压缩包）                          |
|---------|-------------------|---------------------------------------------|---------------------------------------------|
| Linux   | x86_64 (amd64)    | `campus-net-helper_0.1.0_linux-amd64`       | `campus-net-helper_0.1.0_linux-amd64.tar.gz` |
| Linux   | arm64 (aarch64)   | `campus-net-helper_0.1.0_linux-arm64`       | `campus-net-helper_0.1.0_linux-arm64.tar.gz` |
| macOS   | Apple Silicon     | `campus-net-helper_0.1.0_darwin-arm64`      | `campus-net-helper_0.1.0_darwin-arm64.tar.gz`|
| macOS   | Intel x86_64      | `campus-net-helper_0.1.0_darwin-amd64`      | `campus-net-helper_0.1.0_darwin-amd64.tar.gz`|
| Windows | x86_64 (amd64)    | `campus-net-helper_0.1.0_windows-amd64.exe` | `campus-net-helper_0.1.0_windows-amd64.zip` |

压缩包解压后的目录结构：

```
campus-net-helper_0.1.0_linux-amd64/
├── campus-net-helper_0.1.0_linux-amd64    # 可执行文件
├── LICENSE
├── README.md
└── CHANGELOG.md
```

用户选择依据：

* **Linux**：绝大多数桌面与软路由为 x86_64，选 `linux-amd64`；ARM64 设备（如 OpenWrt 网关、树莓派）选 `linux-arm64`。可用 `uname -m` 确认（`x86_64` 或 `aarch64`）。
* **macOS**：Apple Silicon（M1 及以后）选 `darwin-arm64`，Intel 机型选 `darwin-amd64`。可通过「关于本机」的芯片一栏区分。
* **Windows**：常规 64 位系统选 `windows-amd64`，可执行文件为 `campus-net-helper_0.1.0_windows-amd64.exe`。

## 3. 构建流程

### 3.1 前置要求

* Go 1.27+（与 `go.mod` 声明一致）
* `tar`、`zip` 工具（用于打包）

### 3.2 本地构建：`build.sh`

仓库根目录提供 `build.sh`，一条命令产出全部平台的二进制与压缩包：

```sh
./build.sh                # 版本号取 git describe（最近 tag），无 tag 时回退 dev
./build.sh v0.1.0         # 显式指定版本号（推荐，保证产物名与发布版本一致）
```

产物输出到 `dist/`，并附 `sha256sums.txt`：

```
dist/
├── campus-net-helper_0.1.0_linux-amd64.tar.gz
├── campus-net-helper_0.1.0_linux-arm64.tar.gz
├── campus-net-helper_0.1.0_darwin-amd64.tar.gz
├── campus-net-helper_0.1.0_darwin-arm64.tar.gz
├── campus-net-helper_0.1.0_windows-amd64.zip
└── sha256sums.txt
```

脚本使用纯 Go 交叉编译（`CGO_ENABLED=0`），无需目标平台工具链。

### 3.3 自动化构建（推荐用于正式发版）

推 `vX.Y.Z` tag 即自动构建并发布。流程：

1. 检出代码、安装 Go（版本与 `go.mod` 一致）
2. 执行 `./build.sh vX.Y.Z`，产出全部平台二进制
3. 生成 `sha256sums.txt`
4. 调用 `softprops/action-gh-release` 创建 GitHub Release，附带 Release Notes 与全部产物

工作流见 `.github/workflows/release.yml`。

**建议**：正式发布（含跨平台产物齐全）走 GitHub Actions，本地 `build.sh` 仅用于开发自测与验证。若后续需要 macOS/Windows 代码签名、Homebrew formula 等扩展能力，可平滑迁移至 [goreleaser](https://goreleaser.com/)。

### 3.4 手动构建（不依赖脚本）

参照 `build.sh` 的逻辑手动执行即可，核心如下：

```sh
VERSION=v0.1.0
LDFLAGS="-s -w \
  -X main.version=${VERSION#v} \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

mkdir -p dist/build/{linux-amd64,linux-arm64,darwin-amd64,darwin-arm64,windows-amd64}

CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "dist/build/linux-amd64/campus-net-helper_${VERSION#v}_linux-amd64" ./main.go
CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "dist/build/linux-arm64/campus-net-helper_${VERSION#v}_linux-arm64" ./main.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "dist/build/darwin-amd64/campus-net-helper_${VERSION#v}_darwin-amd64" ./main.go
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "dist/build/darwin-arm64/campus-net-helper_${VERSION#v}_darwin-arm64" ./main.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "dist/build/windows-amd64/campus-net-helper_${VERSION#v}_windows-amd64.exe" ./main.go
```

然后按 2 节的目录结构逐个打包为 tar.gz / zip。

## 4. Release Notes 模板

创建 GitHub Release 时按下表填写，语言风格与 README 一致。**必须**列出：用户可见的变更、各平台可用二进制清单、已知问题与限制。

````markdown
# v<版本号>

> <一句话版本摘要。例如：首次正式发布版本，覆盖 Windows / macOS / Linux 全平台。>

## 变更内容

### 新功能
* <示例：新增 macOS Apple Silicon 构建支持>

### 问题修复
* <示例：修复会话过期检测在 IPv6-only 网络环境下的偶发误判>

### 破坏性变更
* <示例：环境变量 `DHU_IFACE` 更名为 `DHU_WAN_IFACE`，旧名称不再受支持>
（如无则省略本节）

## 各平台构建

| 平台    | 架构              | 下载                                          | 校验            |
|---------|-------------------|-----------------------------------------------|-----------------|
| Linux   | x86_64            | `campus-net-helper_<版本号>_linux-amd64.tar.gz`   | sha256sums.txt  |
| Linux   | arm64             | `campus-net-helper_<版本号>_linux-arm64.tar.gz`   | sha256sums.txt  |
| macOS   | Apple Silicon     | `campus-net-helper_<版本号>_darwin-arm64.tar.gz`  | sha256sums.txt  |
| macOS   | Intel             | `campus-net-helper_<版本号>_darwin-amd64.tar.gz`  | sha256sums.txt  |
| Windows | x86_64            | `campus-net-helper_<版本号>_windows-amd64.zip`    | sha256sums.txt  |

将上表中的「`<版本号>`」替换为本次发布版本号（例如 `0.1.0`）。各条下载链接由 GitHub 自动生成。

## 安装说明

各平台安装与开机自启步骤详见 [README](./README.md) 对应章节。

## 快速验证

```sh
./campus-net-helper_0.1.0_linux-amd64 --version
# 预期输出：campus-net-helper 0.1.0 (commit abc1234, built 2026-10-21T12:00:00Z)
```

## 已知问题

* <示例：Windows 上若网卡名称包含非 ASCII 字符，请在 PowerShell 中确认 `DHU_WAN_IFACE` 填写的是系统内部名。>
（如无则写"本次版本无已知问题。"）

## 反馈

* 问题反馈：[Issues](https://github.com/SHORiN-KiWATA/DHUCampusNetHelper/issues)
* 建议与讨论：[Discussions](https://github.com/SHORiN-KiWATA/DHUCampusNetHelper/discussions)（如已启用）
````

## 5. 发布检查清单

发布前逐项确认：

- [ ] `go.mod` 声明的 Go 版本与本地及 CI 构建环境一致
- [ ] `main.go` 已支持 `--version`，可打印注入的 `version` / `commit` / `buildTime`
- [ ] `build.sh` 本地执行成功，`dist/` 下 5 个产物齐全（linux-amd64、linux-arm64、darwin-amd64、darwin-arm64、windows-amd64）
- [ ] 每个二进制在对应平台手动运行一次：登录成功、`--version` 输出正确
- [ ] `sha256sums.txt` 已生成且与全部产物匹配（`sha256sum -c sha256sums.txt`）
- [ ] git tag 与本次发布版本号一致（`vX.Y.Z` 前缀）
- [ ] Release Notes 按模板填写完毕，破坏性变更已单独列出
- [ ] README 中环境变量名、部署步骤与实际版本一致

## 6. 回滚与应急

* 发布后发现严重缺陷：在 GitHub 上将该 Release 设为 `draft`，在仓库 Issues 置顶公告受影响版本，尽快发布对应的 `PATCH` 版本修复。
* 用户侧应急：因二进制自包含、无运行时依赖，用户可直接回退到旧版本二进制继续运行；`build.sh` 支持显式指定版本号，可快速重新构建任意历史版本。
