# S-UI 通用一键安装版

本分支基于 [Teminuosi/s-ui](https://github.com/Teminuosi/s-ui) 和
[Teminuosi/s-ui-frontend](https://github.com/Teminuosi/s-ui-frontend) 构建。

它不是为某个国家、服务商、固定 IP 或特定网络结构定制的。同一套安装代码适用于常见的
Debian、Ubuntu、CentOS、Rocky Linux、AlmaLinux、Fedora、Arch Linux 和 openSUSE VPS。
服务器上具体建立什么节点、是否组成多级网络，由使用者在 S-UI 后台配置，安装器不预设线路。

本仓库是 `gffd01/diandian` 中已验证优化版的独立正式仓库，稳定源码位于 `main`。
原仓库继续保留作为迁移前历史记录，本仓库的安装包和发布流程不再依赖原仓库。

## 这一版解决什么

- 自动安装或升级时生成新的随机后台账号和密码，并清楚显示在终端；
- 安装结束前真实打开登录页、提交生成的账号密码、验证登录后的后台，失败就停止；
- 保留升级前的数据库、节点、面板路径和端口，只有后台账号密码会轮换；
- 自动避开已被其他程序占用的面板端口和订阅端口；
- 下载 Release、校验 SHA-256 后才解压安装；
- 默认只开放后台、订阅和明确填写的节点端口；
- 异常退出自动重启，连接队列按 VPS 内存调整；
- 内核支持时启用 BBR + fq，并启用 TCP MTU 探测；
- API Token 默认 30 天有效，重复升级不会堆积同用途的有效令牌；
- 安装完成后打印服务、拥塞算法、队列、MTU、内存、负载和磁盘诊断结果。

## 一键安装

请用 `root` 用户复制整行执行：

```bash
curl --fail --location --proto '=https' --tlsv1.2 https://raw.githubusercontent.com/gffd01/diandianS-UI/4cbea57ec89270ba96d9c3b41f8da15550cc0907/install.sh -o /tmp/s-ui-universal-install.sh && echo '8b51da54c7b5181a3bd2bbbbeb5e5a84add3f9a220d40b18744a656b12f34d37  /tmp/s-ui-universal-install.sh' | sha256sum --check - && chmod 700 /tmp/s-ui-universal-install.sh && SUI_AUTO=1 /tmp/s-ui-universal-install.sh universal-13
```

这条命令同时固定了新仓库的安装脚本提交、脚本 SHA-256 和 Release 标签。只有出现
`/tmp/s-ui-universal-install.sh: OK` 才会继续安装。`universal-13` 的 7 种 Linux 架构均已通过
GitHub Actions 构建，发布包包含独立的 `SHA256SUMS`，所有下载只来自
`gffd01/diandianS-UI`。

需要先审查源码，可克隆新分支：

```bash
git clone --depth=1 --branch main --single-branch https://github.com/gffd01/diandianS-UI.git
cd diandianS-UI
less install.sh
```

## 默认安装结果

默认是 `direct` 模式：后台监听所有网卡，主机防火墙开放实际后台和订阅端口，安装后可以直接
使用终端打印的地址、随机账号和密码登录。服务商控制台中的安全组不受脚本控制，如外网打不开，
仍需在服务商控制台放行终端显示的后台端口。

如果只允许通过 SSH 隧道管理后台，使用：

```bash
SUI_ACCESS_MODE=tunnel sudo ./install.sh
```

需要脚本同时开放节点端口时，明确列出实际端口：

```bash
SUI_NODE_PORTS=443,8443 sudo ./install.sh
```

完整参数、安全边界和验收方式见 [通用部署说明](SECURE_INSTALL.md)。

## 关于速度

本版会消除主机侧常见的可避免瓶颈，但不会虚假承诺“任何线路都达到同一个最高速度”。实际速度
还受服务商带宽、跨网路由、丢包、共享宿主机、客户端网络和目标站点限制影响。安装器负责把
服务稳定性、TCP 参数和可观测诊断做好；线路本身的问题必须用分段测速和丢包数据判断。

后台的“出站列表”可以分别测量每个出站的延迟和下载吞吐量；“中转管理”可以同时测试中转
出站与 `direct` 出站，帮助判断瓶颈是在 VPS 本身还是中转线路。“全部测速”会动态读取当前全部
落地节点，最多并发 3 个，每个节点下载 5 MiB 真实数据并显示进度、延迟、速度和失败原因。
Hysteria、Brutal 和 TUIC 仍保留兼容默认值，界面会在
默认带宽或拥塞算法可能不适合实际线路时提示用户，不会在缺少线路证据时擅自改写配置。

## 源码构建

GitHub Actions 会先执行脚本语法检查、通用性检查和 Go 测试，再构建前端以及 Linux 静态后端。
构建目标为 `amd64`、`arm64`、`armv7`、`armv6`、`armv5`、`386` 和 `s390x`，全部成功后才发布
带 `SHA256SUMS` 的 `universal-*` Release。

具体上游提交记录在 [`.upstream-versions`](.upstream-versions) 中。前端是仓库内的普通源码，
不依赖外部 Git 子模块。

## 许可

后端沿用仓库中的 GPL-3.0 `LICENSE`。各依赖及前端源码遵循其各自上游许可。
