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

# 查看服务日志（非常重要）
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

> 临时只想手动跑一次的话，也可以在当前终端里用 `$env:DHU_USERNAME="账号"` 等方式临时设置。

### 3. 用任务计划程序开机自启

如果你希望程序在后台一直运行，以保持校园网登录状态，可以用 Windows 自带的「任务计划程序」让它在你每次登录时自动后台执行，分四步：

**第一步：创建任务。**

右键开始菜单，选「终端（管理员）」打开 PowerShell，把整段代码粘贴进去回车执行（`$exe`、`$work` 两行改成你存放 `campus-net-helper.exe` 的实际位置和文件夹，其余原样执行）：

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

执行成功会看到类似 `State : Ready`、`TaskName : campus-net-helper` 的输出，说明任务已创建好了。

**第二步：验证任务能不能正常跑。**

```powershell
# 手动触发一次（相当于以后登录 Windows 时系统会做的那样）
Start-ScheduledTask -TaskName "campus-net-helper"

# 等 5 秒后确认进程真的起来了
Get-Process campus-net-helper
```

能看到 `campus-net-helper.exe` 进程就说明配置成功了，以后每次登录 Windows 它都会自动启动，你不需要再手动开终端。

**第三步：日常操作命令。**

```powershell
# 查看是否正在运行
Get-Process campus-net-helper -ErrorAction SilentlyContinue

# 停止
Stop-Process -Name campus-net-helper

# 查看计划任务状态（State 为 Ready、LastTaskResult 为 0 说明上次运行成功）
Get-ScheduledTask -TaskName campus-net-helper

# 以后不想用了，删除这个计划任务
Unregister-ScheduledTask -TaskName "campus-net-helper" -Confirm:$false
```

> 提示：任务计划程序在 `SYSTEM` 账号下运行，它读到的环境变量是**系统级**的，也就是第 2 步 `setx … /M` 写进去的那些；如果你只是临时在某个终端里用 `$env:` 设过，计划任务里的程序是读不到的，所以第 2 步不要省。

## macOS 用户

### 1. 下载 `campus-net-helper` 二进制

从 `Releases` 页面下载 macOS 平台的 `campus-net-helper`，放进 `PATH` 内的目录，例如 `~/bin/`：

```sh
mkdir -p ~/bin
mv ~/Downloads/campus-net-helper ~/bin/
chmod +x ~/bin/campus-net-helper
```

### 2. 设置环境变量

写入 shell 配置文件，让登录、计划任务时都能读到：

```sh
# zsh 默认读 ~/.zshrc，bash 读 ~/.bash_profile 或 ~/.bashrc
echo 'export DHU_USERNAME="你的账号"'   >> ~/.zshrc
echo 'export DHU_PASSWORD="你的密码"'  >> ~/.zshrc
echo 'export DHU_WAN_IFACE="你的网卡名称"' >> ~/.zshrc
source ~/.zshrc
```

**`DHU_WAN_IFACE` 里填什么？**

这里要填的是 macOS 给网卡起的内部名字，长这样：`en0`、`en1` 这种「`en` + 数字」的形式，不是「Wi-Fi」「有线网络」你在系统设置里看到的名字。查法两种：

- 图形界面：打开「系统设置 → 网络」，点击当前正连着的那个端口进入详情，页面左下角会有一行灰色的 `Hardware: ...`，右侧紧跟的 `en0` 之类的短名就是。
- 终端：执行 `networksetup -listallhardwareports`，输出里每一项有两行，一行是 `Hardware Port: Wi-Fi`（或 `Ethernet`），紧跟着的下一行是 `Device: en0`，这个 `en0` 就是你要填的值。找到你正在用的那种连接方式（连 Wi-Fi 就找 Wi-Fi 那组，插网线就找 Ethernet 那组），把对应的 `Device` 值填进 `DHU_WAN_IFACE`。


### 3. 用 LaunchAgent 登录自启

如果你希望程序在后台一直运行，以保持校园网登录状态，可以用 macOS 的 LaunchAgent 在登录时后台启动：

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

加载并管理：

```sh
# 加载（后台启动）
launchctl load   ~/Library/LaunchAgents/de.shorin.campus-net-helper.plist

# 卸载（停止）
launchctl unload ~/Library/LaunchAgents/de.shorin.campus-net-helper.plist

# 查看状态
launchctl list | grep campus-net-helper

# 查看日志（非常重要）
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

保存后：

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now campus-net-helper   # 启用 + 立即启动
```

日常管理命令：

```sh
sudo systemctl status   campus-net-helper   # 查看服务状态
sudo systemctl restart  campus-net-helper   # 重启
sudo systemctl stop     campus-net-helper   # 停止
sudo journalctl -u      campus-net-helper   # 查看日志（非常重要）
```
