---
title: 快速开始
---

# 快速开始

安装 Docker 和 Docker Compose v2，从 [GitHub 仓库](https://github.com/shyun-cc/lathe-base) 获取源码，再进入源码根目录执行以下命令。

```sh
test -f .env || (umask 077; printf 'LATHE_DB_PASSWORD=%s\n' "$(openssl rand -hex 24)" > .env)
docker compose up --wait
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1:8080/ready
```

端口有自定义配置时，将命令中的 `8080` 换成配置值。启动失败时，在源码根目录运行 `docker compose ps -a` 和 `docker compose logs` 查看服务状态与日志。

如需办理业务，在源码根目录为 API 生成本地令牌。保留已有的 `.env` 内容，并避免重复添加同名配置。

```sh
(umask 077; printf 'LATHE_API_TOKEN=%s\n' "$(openssl rand -hex 32)" >> .env)
docker compose up -d --wait
```

安装 Node.js 24 和 npm，从仓库根目录执行以下命令。

```sh
cd apps/web
npm ci
npm run dev
```

打开终端显示的工作台地址，输入 `.env` 中 `LATHE_API_TOKEN` 的值。如果 API 使用了其他本地端口，例如 8090，以 `LATHE_API_ORIGIN=http://127.0.0.1:8090 npm run dev` 启动工作台。令牌赋予实例的全部业务访问权限，请与数据库密码一同保管。使用共享设备后请退出登录。

按[业务操作顺序](workflow.md)录入和流转单据，或查看[打印与模板配置](printing.md)。

完成试用后，运行 `docker compose down` 停止服务并保留数据卷。参阅[部署指南](deployment.md)了解端口和数据卷设置。
