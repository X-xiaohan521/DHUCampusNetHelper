# 发布流程

## 版本规范

版本号遵循 [语义化版本 2.0.0](https://semver.org/lang/zh-CN/)：

```
MAJOR.MINOR.PATCH[-PRE]
  │     │      │    └─ 预发布标识（可选），如 0.2.0-alpha.1
  │     │      └────── 向下兼容的缺陷修复
  │     └───────────── 向下兼容的新功能
  └─────────────────── 不兼容的变更（如环境变量改名、认证流程变更）
```

发版时打上 git tag（`vX.Y.Z` 前缀），推送后 GitHub Actions 自动构建并创建 Release 草稿：

```sh
git tag v0.1.0
git push origin v0.1.0
```

二进制内嵌版本号由构建时 `-ldflags "-X"` 注入，`./campus-net-helper ... --version` 可查看。

## 二进制命名规范

格式：`campus-net-helper_<版本号>_<GOOS>-<GOARCH>[.exe]`

各平台发布产物（以 `0.1.0` 为例）：

* `campus-net-helper_0.1.0_linux-amd64`
* `campus-net-helper_0.1.0_linux-arm64`
* `campus-net-helper_0.1.0_darwin-amd64`    （macOS Intel）
* `campus-net-helper_0.1.0_darwin-arm64`    （macOS Apple Silicon）
* `campus-net-helper_0.1.0_windows-amd64.exe`

用户选择依据：

* **Linux**：`uname -m` 输出 `x86_64` 选 `linux-amd64`，`aarch64` 选 `linux-arm64`。
* **macOS**：「关于本机」芯片为 Apple M 系列选 `darwin-arm64`，Intel 选 `darwin-amd64`。
* **Windows**：常规 64 位系统选 `windows-amd64`（可执行文件为 `.exe`）。

构建产物附带 `sha256sums.txt`，用户可用 `sha256sum -c sha256sums.txt` 校验文件完整性。

## Release Notes 模板

GitHub Actions 创建 Release 草稿时，会自动根据当前 `git log`（上一个 tag 到当前 tag 之间的提交）生成初始 Notes 并带入版本号。你可以在草稿中补充以下内容，审阅无误后发布：

```markdown
# v<版本号>

> <一句话版本摘要>

## 变更内容
* <来自 git log 的提交摘要，手动整理后保留有意义的条目>

## 各平台构建

| 平台    | 架构          | 文件名                                          |
|---------|---------------|-------------------------------------------------|
| Linux   | x86_64        | `campus-net-helper_<版本号>_linux-amd64`          |
| Linux   | arm64         | `campus-net-helper_<版本号>_linux-arm64`          |
| macOS   | Intel         | `campus-net-helper_<版本号>_darwin-amd64`         |
| macOS   | Apple Silicon | `campus-net-helper_<版本号>_darwin-arm64`         |
| Windows | x86_64        | `campus-net-helper_<版本号>_windows-amd64.exe`    |

## 已知问题
* （如无可省略）

## 反馈
* [Issues](https://github.com/SHORiN-KiWATA/DHUCampusNetHelper/issues)
```

## 发布检查清单

* [ ] 本地 `./build.sh` 运行成功，`dist/` 下 5 个二进制齐全
* [ ] 每个二进制在对应平台运行一次，`--version` 输出正确
* [ ] Release Notes 草稿内容已审阅，版本号一致
