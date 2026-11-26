[简体中文](./README.md) | English | [繁體中文](./README.zh-TW.md)

# Lathe Base

Lathe Base is an open-source ERP foundation for a single factory. It covers master data, sales orders, work orders, outsourced operations, shipments, counterparty reconciliation, offline receipts and disbursements, invoice registration, and configurable print templates.

Version 0.1 uses GoFrame, [shw-gf-lib](https://github.com/shyun-cc/shw-gf-lib), and PostgreSQL 18. The responsive workspace uses Vue 3, TypeScript, and [shw-wui](https://github.com/shyun-cc/shw-wui). It covers master records, sales, production, shipping, reconciliation, and invoice registration, with configurable previews for sales orders, work orders, and delivery notes.

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

Set `LATHE_API_TOKEN` in `.env` before starting or recreating the API service. Install Node.js 24 and npm, run the commands below, then open the displayed URL and sign in with the instance token. See the [user guide](docs/quickstart.md) for custom API ports and same-origin hosting.

```sh
cd apps/web
npm ci
npm run dev
```

## Work on the code

Use Go 1.26 and a C compiler. The shared migration library uses a PostgreSQL parser through CGO.

```sh
gofmt -l apps/api migrations
go vet ./...
go test ./...
```

From `apps/web`, run `npm ci`, `npm run typecheck`, `npm run lint`, and `npm run build` to check the Web app.

Set `LATHE_TEST_DATABASE_URL` to a dedicated PostgreSQL 18 test instance with permission to create databases, then run `go test ./...` to exercise the full workflow. The test creates and drops its own database.

To run outside Compose, set `DATABASE_URL` to your PostgreSQL connection string and optionally set `PORT`. Run `go run ./apps/api/cmd/lathe migrate` before `go run ./apps/api/cmd/lathe serve`.

- [User guide (Simplified Chinese)](docs/index.md)
- [Repository guidelines](AGENTS.md)

Lathe Base is licensed under [Apache License 2.0](LICENSE).
