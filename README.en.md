[简体中文](./README.md) | English | [繁體中文](./README.zh-TW.md)

# Lathe Base

Lathe Base is an open-source ERP foundation for a single factory. It covers master data, sales orders, work orders, outsourced operations, shipments, counterparty reconciliation, offline receipts and disbursements, invoice registration, and configurable print templates.

Version 0.1 milestone 2 provides GoFrame business APIs backed by PostgreSQL 18 and [shw-gf-lib](https://github.com/shyun-cc/shw-gf-lib). Maintain master data and process orders, work order splits, production reports, outsourcing, shipments, reconciliation, offline cash allocation, and invoice registration. The planned responsive Web stack is Vue 3, TypeScript, and [shw-wui](https://github.com/shyun-cc/shw-wui).

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

To use the business API, set `LATHE_API_TOKEN` in `.env` before starting or recreating the API service. Health checks are available without the token. See the [user guide](docs/quickstart.md).

## Work on the code

Use Go 1.26 and a C compiler. The shared migration library uses a PostgreSQL parser through CGO.

```sh
gofmt -l apps/api migrations
go vet ./...
go test ./...
```

Set `LATHE_TEST_DATABASE_URL` to a dedicated PostgreSQL 18 test instance with permission to create databases, then run `go test ./...` to exercise the full workflow. The test creates and drops its own database.

To run outside Compose, set `DATABASE_URL` to your PostgreSQL connection string and optionally set `PORT`. Run `go run ./apps/api/cmd/lathe migrate` before `go run ./apps/api/cmd/lathe serve`.

- [User guide (Simplified Chinese)](docs/index.md)
- [Repository guidelines](AGENTS.md)

Lathe Base is licensed under [Apache License 2.0](LICENSE).
