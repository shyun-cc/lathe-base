---
title: 部署指南
---

# 部署指南

使用仓库自带的 Compose 配置在本机部署。首次启动步骤见[快速开始](quickstart.md)。

- 首次启动会先应用 PostgreSQL 迁移，再启动 API。
- 在 `.env` 中设置 `LATHE_HTTP_PORT` 可更改本机访问端口。备份数据时同时妥善保管该数据卷对应的数据库密码。
- `/health` 检查 API 进程，`/ready` 检查数据库连接。

维护已有实例时，先备份 PostgreSQL，再按目标版本的升级说明操作。停止实例时使用 `docker compose down`，这样可以保留数据卷。

## 同源托管工作台

拥有运行 Go API 的环境时，可由 API 同源服务已构建的工作台。使用 Go 1.26、C 编译器、Node.js 24 和 npm，在源码根目录执行以下命令。

```sh
npm --prefix apps/web ci
npm --prefix apps/web run build
go build -o bin/lathe ./apps/api/cmd/lathe
```

将 `bin/lathe` 和 `apps/web/dist` 拷贝到部署位置。通过进程环境设置 `DATABASE_URL`、`LATHE_API_TOKEN`、`LATHE_WEB_ROOT`（指向构建后的 `dist` 目录），以及所需的 `PORT`（默认 8080）。升级时先备份数据库并执行 `lathe migrate`，然后用 `lathe serve` 启动。打开服务根地址进入工作台，远程访问应使用 HTTPS。
