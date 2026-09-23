# DHU 校园网自动登录程序
## 功能介绍
* 自动检测网络
* 自动登录

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

启用开机自启动（这一步至关重要）：

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