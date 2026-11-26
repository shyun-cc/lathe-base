package main

import (
	"context"
	"database/sql"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/shyun-cc/shw-gf-lib/migrate"

	"github.com/shyun-cc/lathe-base/apps/api/internal/apihttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/invoice"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/master"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/outsource"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/production"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/sales"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/settlement"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/shipping"
	"github.com/shyun-cc/lathe-base/migrations"
)

func bindBusiness(server *ghttp.Server, db *sql.DB, token string) {
	server.SetClientMaxBodySize(1 << 20)
	server.Group("/api/v1", func(group *ghttp.RouterGroup) {
		group.Middleware(apihttp.Authorize(token))
		master.Bind(group, db)
		sales.Bind(group, db)
		production.Bind(group, db)
		outsource.Bind(group, db)
		shipping.Bind(group, db)
		settlement.Bind(group, db)
		invoice.Bind(group, db)
	})
}

func migrateUp(ctx context.Context, db *sql.DB) error {
	runner, err := migrate.New(migrate.Config{
		FS:      migrations.Files,
		Dialect: migrate.DialectPostgres,
		DB:      func() (*sql.DB, error) { return db, nil },
	})
	if err != nil {
		return err
	}
	return runner.Up(ctx)
}
