# DHU 校园网 - 自动登录助手
> 一款主要面向 **桌面端 和 OpenWrt** 的 DHU 校园网认证助手，让你免去手动登录校园网的麻烦。

## 功能特点
* **自动认证** 校园网
* 监测网络状态，认证 **过期自动重新认证**
* **OpenWrt 软路由** 可用
* 自动区分 **认证过期** 和 **网络故障**，不做无效重试
* 走正常认证流程，**无封号风险**

## 在 OpenWrt 上部署
### 1. 安装 `campus-net-helper` 二进制

从 `Releases` 页面下载本软件二进制包，放在 `/usr/local/bin/` 目录下（如果没有该目录，则手动创建）。

接着，设置程序为可执行：

```sh
chmod 755 /usr/local/bin/campus-net-helper
```

### 2. 添加环境变量

为了使程序能在系统的各个位置通过 `campus-net-helper` 命令调用，需要将 /usr/local/bin 添加到环境变量 `PATH` 中。

编辑 `/etc/profile` 文件，找到这一行：

```profile
export PATH="/usr/sbin:/usr/bin:/sbin:/bin"
```

在后面添加 `:/usr/local/bin`（如果已有则不用添加），即：

```profile
export PATH="/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/bin"
```

然后，在当前 shell 中执行：

```sh
source /etc/profile
```

等待 shell 刷新后，执行 `campus-net-helper` 命令即可启动校园网自动登录程序。

### 3. 注册为系统进程

上一步完成后，手动执行 `campus-net-helper` 可以完成校园网登录，但如果当前 shell 退出，则程序一并退出，后续无法自动检测网络断开并重新认证。所以，需要将程序注册为系统进程，以便后续持续监测网络状态并保持校园网登录。

OpenWrt 使用 procd 维护系统进程，想要把校园网登录程序变为系统进程，需要编辑配置文件。
编辑或创建 `/etc/init.d/campus-net-helper` 脚本，写入（记得替换你的账号、密码和用于连接校园网的网卡名称）：

```campus-net-helper
#!/bin/sh /etc/rc.common

# Managing service with procd.
USE_PROCD=1

# Define the priority of the service.
# The larger the number of START, the later the service is started.
# Make sure to wait until the network is ready before starting this service.
START=95
STOP=10

start_service() {
    procd_open_instance "campus-net-helper"

    # Must use the absolute path to the binary.
    procd_set_param command /usr/local/bin/campus-net-helper

    # Set environment variables.
    procd_set_param env \
        DHU_USERNAME="你的账号" \
        DHU_PASSWORD="你的密码" \
        DHU_WAN_IFACE="用于连接校园网的网卡名称"

    # [Important] Set the daemon to restart the process when it quit.
    # threshold(3600s) timeout(5s) retry(50times)
    procd_set_param respawn 3600 5 50

    # Redirect the stdout and stderr to system journal.
    procd_set_param stdout 1
    procd_set_param stderr 1

    procd_close_instance
}

# If the program supports hot reload, define reload_service here.
# reload_service() {
#     procd_send_signal myportal
# }
```

保存后，赋予脚本可执行权限：

```sh
chmod 755 /etc/init.d/campus-net-helper
```

启用开机自启动：

```sh
/etc/init.d/campus-net-helper enable
```

执行 enable 后，系统会在 /etc/rc.d/ 目录下创建一个指向脚本的符号链接（如 S95campus-net-helper），确保它在每次开机时自动启动。

立即启动服务：

```sh
/etc/init.d/campus-net-helper start
```

之后，你就可以像管理系统服务一样管理它了：

```sh
# 查看服务状态
/etc/init.d/campus-net-helper status

# 重启服务
/etc/init.d/campus-net-helper restart

# 停止服务
/etc/init.d/campus-net-helper stop

# 查看服务日志
logread | grep campus-net-helper
```

## Windows 用户

### 1. 下载 `campus-net-helper.exe` 二进制

从 `Releases` 页面下载 Windows 平台的 `campus-net-helper.exe`，解压到任意目录，例如 `C:\Tools\campus-net-helper\`。

### 2. 设置环境变量

将账号、密码和网卡名设置为系统环境变量（这样程序在任意目录运行、开机自启时都能读到）：

```powershell
# 以管理员身份打开 PowerShell 执行（替换为你的实际值）
setx DHU_USERNAME "你的账号" /M
setx DHU_PASSWORD "你的密码" /M
setx DHU_WAN_IFACE "你的网卡名称" /M   # 连接校园网的那块网卡
```

设置完成后，重新打开一个终端，执行 `campus-net-helper.exe` 即可登录。

> 若仅需在当前会话中临时运行，也可以直接在终端里通过 `$env:DHU_USERNAME="账号"` 等方式进行临时设置。

### 3. 用任务计划程序开机自启

如果你希望程序在后台一直运行，以保持校园网登录状态，可以用 Windows 自带的「任务计划程序」让它在你每次登录时自动后台执行，分四步：

**第一步：创建任务。**

右键开始菜单，选择「终端（管理员）」打开 PowerShell，然后将整段代码粘贴进终端执行（`$exe`、`$work` 两行需更换为你实际存放 `campus-net-helper.exe` 的位置与所在文件夹，其余保持不变）：

```powershell
# ↓↓ 这两行改成你的实际路径 ↓↓
$exe  = "C:\Tools\campus-net-helper\campus-net-helper.exe"
$work = "C:\Tools\campus-net-helper"

# ↓↓↓ 以下不用改，整段一起执行 ↓↓↓
$action    = New-ScheduledTaskAction -Execute $exe -WorkingDirectory $work
$trigger   = New-ScheduledTaskTrigger -AtLogOn
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -RunLevel Highest
$settings  = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries

Register-ScheduledTask -TaskName "campus-net-helper" `
  -Action $action -Trigger $trigger -Principal $principal -Settings $settings
```

若输出中包含 `State : Ready`、`TaskName : campus-net-helper` 等字段，则表明任务已创建完成。

**第二步：验证任务能够正常启动。**

```powershell
# 手动触发一次，效果等同于登录 Windows 时系统自动启动该任务
Start-ScheduledTask -TaskName "campus-net-helper"

# 约 5 秒后确认进程已启动
Get-Process campus-net-helper
```

若能看到 `campus-net-helper.exe` 进程，则表明配置已生效。此后每次登录 Windows，程序都会自动启动，无需再手动打开终端。

**第三步：日常操作命令。**

```powershell
# 查看是否正在运行
Get-Process campus-net-helper -ErrorAction SilentlyContinue

# 停止
Stop-Process -Name campus-net-helper

# 查看计划任务状态（State 为 Ready、LastTaskResult 为 0 说明上次运行成功）
Get-ScheduledTask -TaskName campus-net-helper

# 若不再需要，可删除该计划任务
Unregister-ScheduledTask -TaskName "campus-net-helper" -Confirm:$false
```

> 说明：任务计划程序以 `SYSTEM` 账号运行，其可读取的仅为**系统级**环境变量，即第 2 步 `setx … /M` 写入的变量；若仅在单个终端中通过 `$env:` 临时设置，则计划任务内的程序无法读取这些变量，因此请务必完成第 2 步。

## macOS 用户

### 1. 下载 `campus-net-helper` 二进制

从 `Releases` 页面下载 macOS 平台的 `campus-net-helper`，放进 `PATH` 内的目录，例如 `~/bin/`：

```sh
mkdir -p ~/bin
mv ~/Downloads/campus-net-helper ~/bin/
chmod +x ~/bin/campus-net-helper
```

### 2. 设置环境变量

写入 shell 配置文件，以便登录及后台任务均能读取：

```sh
# zsh 默认读 ~/.zshrc，bash 读 ~/.bash_profile 或 ~/.bashrc
echo 'export DHU_USERNAME="你的账号"'   >> ~/.zshrc
echo 'export DHU_PASSWORD="你的密码"'  >> ~/.zshrc
echo 'export DHU_WAN_IFACE="你的网卡名称"' >> ~/.zshrc
source ~/.zshrc
```

**关于 `DHU_WAN_IFACE` 的填写方式**

此处应填写 macOS 为网卡分配的内部名称，其形式为 `en` 加数字，例如 `en0`、`en1`，而非「Wi-Fi」「有线网络」等系统设置中显示的端口名称。可用以下两种方式查询：

- 图形界面：打开「系统设置 → 网络」，点击当前正在使用的端口进入详情页，其左下角会以灰色小字显示 `Hardware: ...`，紧跟其后的 `en0` 等短名即为所需值。
- 终端：执行 `networksetup -listallhardwareports`，输出中每个端口条目包含两行，第一行形如 `Hardware Port: Wi-Fi`（或 `Ethernet`），紧随其后的 `Device: en0` 中的值即为所需值。请根据你实际的连接方式（Wi-Fi 或有线）找到对应条目，将其 `Device` 值填入 `DHU_WAN_IFACE`。


### 3. 用 LaunchAgent 登录自启

若需让程序在后台持续运行以保持校园网登录状态，可借助 macOS 的 LaunchAgent 机制，在用户登录时自动于后台启动：

```sh
mkdir -p ~/Library/LaunchAgents
cat > ~/Library/LaunchAgents/de.shorin.campus-net-helper.plist <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key>
  <string>de.shorin.campus-net-helper</string>
  <key>ProgramArguments</key>
  <array>
    <string>/bin/zsh</string>
    <string>-lc</string>
    <string>source ~/.zshrc &amp;&amp; exec ~/bin/campus-net-helper</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>~/Library/Logs/campus-net-helper.log</string>
  <key>StandardErrorPath</key>
  <string>~/Library/Logs/campus-net-helper.err.log</string>
</dict></plist>
EOF
```

加载并进行日常管理：

```sh
# 加载（后台启动）
launchctl load   ~/Library/LaunchAgents/de.shorin.campus-net-helper.plist

# 卸载（停止）
launchctl unload ~/Library/LaunchAgents/de.shorin.campus-net-helper.plist

# 查看状态
launchctl list | grep campus-net-helper

# 查看日志
tail -f ~/Library/Logs/campus-net-helper.log
```

## Linux 用户

适用于带 systemd 的发行版（Ubuntu / Debian / Arch / Fedora 等）。

### 1. 安装 `campus-net-helper` 二进制

从 `Releases` 页面下载 Linux 平台二进制，放到 `/usr/local/bin/`：

```sh
sudo mv ~/Downloads/campus-net-helper /usr/local/bin/
sudo chmod 755 /usr/local/bin/campus-net-helper
```

先手动确认脚本能正常运行：

```sh
DHU_USERNAME=账号 DHU_PASSWORD=密码 DHU_WAN_IFACE=网卡名 \
  /usr/local/bin/campus-net-helper
```

### 2. 创建 systemd 服务

```ini
# /etc/systemd/system/campus-net-helper.service

[Unit]
Description=DHU Campus Net Helper
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=DHU_USERNAME=你的账号
Environment=DHU_PASSWORD=你的密码
Environment=DHU_WAN_IFACE=你的网卡名称
ExecStart=/usr/local/bin/campus-net-helper
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

保存后执行：

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now campus-net-helper   # 启用 + 立即启动
```

以下为日常管理命令：

```sh
sudo systemctl status   campus-net-helper   # 查看服务状态
sudo systemctl restart  campus-net-helper   # 重启
sudo systemctl stop     campus-net-helper   # 停止
sudo journalctl -u      campus-net-helper   # 查看日志
```
