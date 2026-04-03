# OpsTunnel

跨平台 SSH 隧道管理器，支持桌面应用、Web UI 和 Docker 部署。

管理 SSH 连接，通过多跳 SSH 链创建 Local (-L) / Remote (-R) / Dynamic SOCKS5 (-D) 隧道，实时监控与自动重连。

## 功能特性

- **SSH 连接管理** - 创建、测试、复用 SSH 连接
- **多跳 SSH 链** - 支持多台跳板机串联，拖拽排序
- **三种隧道模式** - 本地转发 (-L)、远程转发 (-R)、动态 SOCKS5 代理 (-D，支持 CONNECT + BIND)
- **自动重连** - 指数退避重试、速率限制、优雅停止
- **桌面应用** - 原生窗口 + 系统托盘（彩色图标、隧道菜单、一键复制地址）
- **Web UI** - 类 APP 界面，浏览器直接访问
- **Docker** - 多阶段构建 distroless 镜像，一键部署
- **实时监控** - WebSocket 驱动的状态更新和日志流
- **设置中心** - 主题切换（浅色/深色/跟随系统）、日志级别、版本检查
- **API 安全** - Bearer Token 认证、CORS、请求体大小限制

## 快速开始

### 桌面应用

从 [Releases](https://github.com/maxzhang666/ops-tunnel/releases) 下载最新版本并运行。

### Docker

```bash
docker run -d --name ops-tunnel \
  -p 9876:9876 \
  -v tunnel-data:/data \
  ghcr.io/maxzhang666/ops-tunnel:latest
```

打开 http://localhost:9876

### Docker Compose

```bash
curl -O https://raw.githubusercontent.com/maxzhang666/ops-tunnel/main/docker-compose.yml
docker compose up -d
```

### 服务器二进制

```bash
./tunnel-server --listen 127.0.0.1:9876 --data-dir ./data
```

环境变量：`TUNNEL_LISTEN`、`TUNNEL_DATA_DIR`、`TUNNEL_TOKEN`

## 开发

```bash
# 环境要求：Go 1.26+、Node 22+、pnpm

# 安装前端依赖
make install-ui

# 启动开发模式（服务器 + 前端）
make dev

# 启动桌面应用
make dev-desktop

# 构建
make build                       # 服务器 + 前端
VERSION=1.0.0 make build-desktop # 带版本号的桌面应用
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.26, chi/v5, golang.org/x/crypto/ssh |
| 前端 | React 19, TypeScript, Vite, Tailwind 4, shadcn/ui |
| 桌面 | Wails v2, fyne.io/systray |
| 状态管理 | TanStack Query v5, WebSocket |
| CI/CD | GitHub Actions, GHCR, Docker 多阶段构建 |

## 项目结构

```
cmd/
  tunnel-server/    无头 HTTP+WS API 服务器
  tunnel-desktop/   Wails 桌面应用（含系统托盘）
internal/
  config/           数据模型、校验、文件持久化
  ssh/              SSH 认证、主机密钥、链式连接、心跳
  engine/           隧道监管器、退避重试、事件总线
  forward/          本地/远程/动态转发器实现
  api/              HTTP API、WebSocket、中间件
ui/                 React 单页应用
```

## API

默认端口：`9876`

```
GET    /healthz
GET    /ws                                WebSocket 事件流

/api/v1:
  GET/POST       /ssh-connections          SSH 连接 CRUD
  GET/PUT/PATCH/DELETE /ssh-connections/{id}
  POST           /ssh-connections/{id}/test 测试连接
  GET/POST       /tunnels                  隧道 CRUD
  GET/PUT/PATCH/DELETE /tunnels/{id}
  POST           /tunnels/{id}/start|stop|restart 控制
  GET            /tunnels/{id}/status       状态查询
  GET/PATCH      /settings                 设置
  GET            /version                  版本信息
```

## 许可证

MIT
