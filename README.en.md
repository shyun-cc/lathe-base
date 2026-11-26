[简体中文](./README.md) | English | [繁體中文](./README.zh-TW.md)

# Lathe Base

Lathe Base is an open-source ERP foundation for a single factory. It covers master data, sales orders, work orders, outsourced operations, shipments, counterparty reconciliation, offline receipts and disbursements, invoice registration, and configurable print templates.

Version 0.1 milestone 1 provides a modular Go and GoFrame API, PostgreSQL schema migrations using [shw-gf-lib](https://github.com/shyun-cc/shw-gf-lib), and tested rules for splitting work orders and allocating offline cash entries. The running API exposes health and readiness probes. Business screens and write APIs are planned for later milestones. The responsive Web stack is Vue 3, TypeScript, and [shw-wui](https://github.com/shyun-cc/shw-wui).

Financial scope covers operational receivables and payables, reconciliation statements, offline cash allocation, and invoice records. Payment processing, a general ledger, and electronic invoice issuance are outside the scope of v0.1.

## Run locally

Use Docker with the Compose plugin. Create a local environment file once, then start the stack.

```sh
test -f .env || (umask 077; printf 'LATHE_DB_PASSWORD=%s\n' "$(openssl rand -hex 24)" > .env)
docker compose up --wait
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1:8080/ready
```

Compose applies the embedded migrations before starting the API. `/health` reports process health and `/ready` checks PostgreSQL connectivity. Set `LATHE_HTTP_PORT` in `.env` to use another local port. `docker compose down` stops the stack and keeps the database volume. Keep the generated local password with that volume.

## Work on the code

Use Go 1.26 and a C compiler. The shared migration library uses a PostgreSQL parser through CGO.

```sh
gofmt -l apps/api migrations
go vet ./...
go test ./...
```

To run outside Compose, set `DATABASE_URL` to your PostgreSQL connection string and optionally set `PORT`. Run `go run ./apps/api/cmd/lathe migrate` before `go run ./apps/api/cmd/lathe serve`.

- [User guide (Simplified Chinese)](docs/index.md)
- [Repository guidelines](AGENTS.md)

Lathe Base is licensed under [Apache License 2.0](LICENSE).
