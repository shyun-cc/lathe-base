package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/shyun-cc/shw-gf-lib/response"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) != 1 || (args[0] != "serve" && args[0] != "migrate") {
		return errors.New("usage: lathe {serve|migrate}")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return errors.New("invalid PostgreSQL connection configuration")
	}
	defer db.Close()

	if args[0] == "migrate" {
		return migrateUp(ctx, db)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if n, err := strconv.ParseUint(port, 10, 16); err != nil || n == 0 {
		return fmt.Errorf("PORT must be an integer between 1 and 65535")
	}
	addr := net.JoinHostPort("", port)
	server := ghttp.GetServer("lathe")
	server.SetAddr(addr)
	server.BindHandler("GET:/health", func(r *ghttp.Request) {
		response.Success(r, map[string]string{"status": "ok"})
	})
	server.BindHandler("GET:/ready", func(r *ghttp.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			r.Response.WriteHeader(http.StatusServiceUnavailable)
			r.Response.WriteJsonExit(map[string]string{"status": "unavailable"})
		}
		response.Success(r, map[string]string{"status": "ready"})
	})
	bindBusiness(server, db, os.Getenv("LATHE_API_TOKEN"))
	if err := server.Start(); err != nil {
		return err
	}
	ghttp.Wait()
	return nil
}
