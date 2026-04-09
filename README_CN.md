# OpsTunnel

跨平台 SSH 隧道管理器，通过可视化界面创建、监控和自动重连 SSH 隧道 — 支持桌面应用、Web UI 和 Docker 部署。

## 截图

<!-- TODO: 添加截图 -->

![仪表盘](docs/screenshots/dashboard.png)

![隧道管理](docs/screenshots/tunnels.png)

![隧道信息](docs/screenshots/tunnel-info.png)

![SSH 连接](docs/screenshots/ssh-connections.png)

## 功能特性

- **多跳隧道链** — 通过一台或多台跳板机访问内网数据库、API 或其他服务
- **SOCKS5 代理** — 通过动态隧道浏览内网，支持用户名密码认证和 IP 黑白名单
- **三种隧道模式** — 本地转发 (-L)、远程转发 (-R)、动态 SOCKS5 (-D)
- **自动重连** — 连接断开后自动恢复，支持配置重试策略
- **流量仪表盘** — 实时带宽图表、各隧道流量统计、连接数监控
- **登录保护** — Server/Docker 模式下可开启登录页，也支持 Token 方式调用 API
- **桌面应用** — 原生窗口 + 系统托盘图标，一眼查看隧道状态
- **Docker 部署** — 一条命令启动，数据持久化
- **多语言** — 支持英文和简体中文

## 安装

### 桌面应用

从 [Releases](https://github.com/maxzhang666/ops-tunnel/releases) 下载适合你平台的最新版本。

### Docker

```bash
docker run -d --name ops-tunnel \
  -p 9876:9876 \
  -v tunnel-data:/data \
  -e TUNNEL_ADMIN_USERNAME=admin \
  -e TUNNEL_ADMIN_PASSWORD=your-password \
  ghcr.io/maxzhang666/ops-tunnel:latest
```

打开 http://localhost:9876 ，使用设置的用户名和密码登录。

### Docker Compose

```bash
curl -O https://raw.githubusercontent.com/maxzhang666/ops-tunnel/main/docker-compose.yml
docker compose up -d
```

### 服务器二进制

```bash
./tunnel-server --listen 127.0.0.1:9876 --data-dir ./data
```

## 配置

支持命令行参数和环境变量两种方式，环境变量在未指定对应参数时生效。

| 环境变量 | 命令行参数 | 默认值 | 说明 |
|---------|-----------|-------|------|
| `TUNNEL_LISTEN` | `--listen` | `127.0.0.1:9876` | HTTP 监听地址 |
| `TUNNEL_DATA_DIR` | `--data-dir` | `./data` | 数据目录 |
| `TUNNEL_UI_DIR` | `--ui-dir` | (内嵌) | 自定义前端文件路径 |
| `TUNNEL_TOKEN` | `--token` | (无) | API 调用凭证 |
| `TUNNEL_ADMIN_PASSWORD` | — | (无) | 管理后台密码，不设置则免登录 |
| `TUNNEL_ADMIN_USERNAME` | — | `admin` | 管理后台用户名 |

不设置 `TUNNEL_ADMIN_PASSWORD` 时，打开页面直接可用，无需登录。

## 许可证

MIT
