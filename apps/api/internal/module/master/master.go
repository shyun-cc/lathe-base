package master

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/apihttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

type record struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Active     *bool  `json:"active"`
	IsCustomer bool   `json:"is_customer"`
	IsSupplier bool   `json:"is_supplier"`
	UnitID     int64  `json:"unit_id"`
}

type resource struct {
	db      *sql.DB
	table   string
	key     string
	columns string
}

func Bind(group *ghttp.RouterGroup, db *sql.DB) {
	group.GET("/customers", apihttp.List(store.List(db, `SELECT id, code, name, active, is_customer, is_supplier
		FROM master_parties WHERE id > $1 AND is_customer ORDER BY id LIMIT $2`)))
	group.GET("/suppliers", apihttp.List(store.List(db, `SELECT id, code, name, active, is_customer, is_supplier
		FROM master_parties WHERE id > $1 AND is_supplier ORDER BY id LIMIT $2`)))
	for _, definition := range []struct{ path, table, key, extra string }{
		{"/parties", "master_parties", "code", ", is_customer, is_supplier"},
		{"/units", "master_units", "code", ""},
		{"/products", "master_products", "sku", ", unit_id"},
		{"/operations", "master_operations", "code", ""},
		{"/employees", "master_employees", "code", ""},
	} {
		res := resource{db: db, table: definition.table, key: definition.key,
			columns: "id, " + definition.key + " AS code, name, active" + definition.extra}
		group.GET(definition.path, apihttp.List(res.list))
		group.POST(definition.path, apihttp.Body(res.create))
		group.GET(definition.path+"/:id", apihttp.ID(res.get))
		group.PUT(definition.path+"/:id", apihttp.BodyID(res.update))
		group.DELETE(definition.path+"/:id", apihttp.ID(res.delete))
	}
}

func (r resource) list(ctx context.Context, after, limit int64) (json.RawMessage, error) {
	return store.JSON(ctx, r.db, "SELECT coalesce(json_agg(item ORDER BY id), '[]') FROM (SELECT "+
		r.columns+" FROM "+r.table+" WHERE id > $1 ORDER BY id LIMIT $2) item", after, limit)
}

func (r resource) get(ctx context.Context, id int64) (json.RawMessage, error) {
	return store.JSON(ctx, r.db, "SELECT row_to_json(item) FROM (SELECT "+
		r.columns+" FROM "+r.table+" WHERE id = $1) item", id)
}

func (r resource) fields(input record) ([]string, []any, error) {
	code, name := strings.TrimSpace(input.Code), strings.TrimSpace(input.Name)
	if code == "" || name == "" {
		return nil, nil, store.Invalid("code and name are required")
	}
	fields := []string{r.key, "name"}
	values := []any{code, name}
	if input.Active != nil {
		fields = append(fields, "active")
		values = append(values, *input.Active)
	}
	switch r.table {
	case "master_parties":
		if !input.IsCustomer && !input.IsSupplier {
			return nil, nil, store.Invalid("enable at least one party role")
		}
		fields = append(fields, "is_customer", "is_supplier")
		values = append(values, input.IsCustomer, input.IsSupplier)
	case "master_products":
		if input.UnitID <= 0 {
			return nil, nil, store.Invalid("unit_id must be positive")
		}
		fields = append(fields, "unit_id")
		values = append(values, input.UnitID)
	}
	return fields, values, nil
}

func marker(index int) string { return "$" + strconv.Itoa(index) }

func (r resource) create(ctx context.Context, input record) (store.ID, error) {
	fields, values, err := r.fields(input)
	if err != nil {
		return store.ID{}, err
	}
	markers := make([]string, len(fields))
	for i := range markers {
		markers[i] = marker(i + 1)
	}
	return store.Insert(ctx, r.db, "INSERT INTO "+r.table+" ("+strings.Join(fields, ", ")+
		") VALUES ("+strings.Join(markers, ", ")+") RETURNING id", values...)
}

func (r resource) update(ctx context.Context, id int64, input record) (store.ID, error) {
	fields, values, err := r.fields(input)
	if err != nil {
		return store.ID{}, err
	}
	assignments := make([]string, len(fields))
	for i, field := range fields {
		assignments[i] = field + " = " + marker(i+1)
	}
	values = append(values, id)
	return store.Insert(ctx, r.db, "UPDATE "+r.table+" SET "+strings.Join(assignments, ", ")+
		" WHERE id = "+marker(len(values))+" RETURNING id", values...)
}

func (r resource) delete(ctx context.Context, id int64) (store.ID, error) {
	return store.Insert(ctx, r.db, "DELETE FROM "+r.table+" WHERE id = $1 RETURNING id", id)
}
