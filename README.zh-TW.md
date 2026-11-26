[简体中文](./README.md) | [English](./README.en.md) | 繁體中文

# 母版 · Lathe Base

母版是為單一工廠、單一執行個體設計的開源 ERP 基礎，涵蓋基礎資料、銷售訂單、生產工單、委外加工、出貨、往來對帳、線下收付款、發票登錄與可設定的列印範本。

v0.1 里程碑 2 提供由 GoFrame 和 [shw-gf-lib](https://github.com/shyun-cc/shw-gf-lib) 支援的 PostgreSQL 18 業務 API。可維護基礎資料，並依序處理銷售訂單、拆單、報工、委外、出貨、往來對帳、線下收支沖銷及發票登錄。響應式 Web 預計採用 Vue 3、TypeScript 與 [shw-wui](https://github.com/shyun-cc/shw-wui)。

財務範圍涵蓋業務應收應付、往來對帳單、線下收支沖銷與發票登錄。支付服務、會計總帳與電子發票開立均在 v0.1 範圍之外。

## 本機啟動

使用含 Compose 外掛的 Docker。第一次啟動時先產生本機設定檔，再啟動服務。

```sh
test -f .env || (umask 077; printf 'LATHE_DB_PASSWORD=%s\n' "$(openssl rand -hex 24)" > .env)
docker compose up --wait
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1:8080/ready
```

Compose 會在 API 啟動前套用內嵌遷移。`/health` 檢查 API 程序，`/ready` 檢查 PostgreSQL 連線。可在 `.env` 中設定 `LATHE_HTTP_PORT` 以更換本機連接埠。`docker compose down` 會停止服務並保留資料庫磁碟區。請保留該磁碟區使用的本機密碼。

若要試用業務 API，先在 `.env` 設定 `LATHE_API_TOKEN`，再啟動或重新建立 API 服務。健康檢查無需權杖。詳見[使用手冊](docs/quickstart.md)。

## 開發

需要 Go 1.26 和 C 編譯器。共用遷移函式庫透過 CGO 使用 PostgreSQL 剖析器。

```sh
gofmt -l apps/api migrations
go vet ./...
go test ./...
```

將 `LATHE_TEST_DATABASE_URL` 指向具建庫權限的隔離 PostgreSQL 18 測試環境，再執行 `go test ./...` 驗證完整流程。測試會建立並刪除自己使用的資料庫。

若在 Compose 以外執行，請將 `DATABASE_URL` 設定為 PostgreSQL 連線字串，也可透過 `PORT` 設定 API 連接埠。先執行 `go run ./apps/api/cmd/lathe migrate`，再執行 `go run ./apps/api/cmd/lathe serve`。

- [使用手冊（簡體中文）](docs/index.md)
- [版本庫協作規範](AGENTS.md)

母版採用 [Apache License 2.0](LICENSE) 開源授權。
