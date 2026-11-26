简体中文 | [English](./README.en.md) | [繁體中文](./README.zh-TW.md)

# 母版 · Lathe Base

母版是面向单厂单实例的开源 ERP 底座，覆盖基础档案、销售订单、生产工单、委外加工、发货、往来对账、线下收付款、发票登记与可配置打印模板。

v0.1 使用 GoFrame、[shw-gf-lib](https://github.com/shyun-cc/shw-gf-lib) 和 PostgreSQL 18。响应式工作台使用 Vue 3、TypeScript 和 [shw-wui](https://github.com/shyun-cc/shw-wui)，可处理从基础档案、销售订单到生产、发货、往来结算和发票登记，并预览和打印销售单、工单和送货单。

财务范围包含业务应收应付、往来对账单、线下收支核销和发票登记。支付通道、会计总账和电子发票开具属于 v0.1 范围之外。

## 本地启动

使用带 Compose 插件的 Docker。首次启动先生成本地配置，再启动服务。

```sh
test -f .env || (umask 077; printf 'LATHE_DB_PASSWORD=%s\n' "$(openssl rand -hex 24)" > .env)
docker compose up --wait
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1:8080/ready
```

Compose 会在 API 启动前应用内嵌迁移。`/health` 检查 API 进程，`/ready` 检查 PostgreSQL 连接。可以在 `.env` 中设置 `LATHE_HTTP_PORT` 更换本地端口。`docker compose down` 会停止服务并保留数据库卷。保留与该数据库卷对应的本地密码。

试用工作台前，在 `.env` 中配置 `LATHE_API_TOKEN` 并启动或重新创建 API 服务。需要 Node.js 24 和 npm。启动以下命令后，访问终端显示的地址并输入实例令牌。自定义 API 端口及同源部署见[用户使用手册](docs/quickstart.md)。

```sh
cd apps/web
npm ci
npm run dev
```

## 开发

需要 Go 1.26 和 C 编译器。公共迁移库通过 CGO 使用 PostgreSQL 解析器。

```sh
gofmt -l apps/api migrations
go vet ./...
go test ./...
```

前端检查在 `apps/web` 下运行 `npm ci`、`npm run typecheck`、`npm run lint` 和 `npm run build`。

将 `LATHE_TEST_DATABASE_URL` 指向有建库权限的隔离 PostgreSQL 18 测试实例，再执行 `go test ./...` 运行闭环集成测试。测试创建并删除自己使用的数据库。

在 Compose 之外运行时，将 `DATABASE_URL` 设置成 PostgreSQL 连接字符串，可通过 `PORT` 配置 API 端口。先执行 `go run ./apps/api/cmd/lathe migrate`，再执行 `go run ./apps/api/cmd/lathe serve`。

- [用户使用手册](docs/index.md)
- [仓库协作规范](AGENTS.md)

母版以 [Apache License 2.0](LICENSE) 开源。
