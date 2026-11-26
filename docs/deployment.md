---
title: 部署指南
---

# 部署指南

使用仓库自带的 Compose 配置在本机部署。首次启动步骤见[快速开始](quickstart.md)。

- 首次启动会先应用 PostgreSQL 迁移，再启动 API。
- 在 `.env` 中设置 `LATHE_HTTP_PORT` 可更改本机访问端口。备份数据时同时妥善保管该数据卷对应的数据库密码。
- `/health` 检查 API 进程，`/ready` 检查数据库连接。

维护已有实例时，先备份 PostgreSQL，再按目标版本的升级说明操作。停止实例时使用 `docker compose down`，这样可以保留数据卷。
