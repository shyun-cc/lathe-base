package invoice

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/apihttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

type registerInput struct {
	PartyID          int64  `json:"party_id"`
	Direction        string `json:"direction"`
	InvoiceNo        string `json:"invoice_no"`
	InvoicedOn       string `json:"invoiced_on"`
	Currency         string `json:"currency"`
	GrossAmountMinor int64  `json:"gross_amount_minor"`
	TaxAmountMinor   int64  `json:"tax_amount_minor"`
	StatementID      *int64 `json:"statement_id"`
	Notes            string `json:"notes"`
}

type service struct{ db *sql.DB }

func Bind(group *ghttp.RouterGroup, db *sql.DB) {
	s := service{db}
	group.POST("/invoices", apihttp.Body(s.register))
	group.GET("/invoices", apihttp.List(s.list))
	group.GET("/invoices/:id", apihttp.ID(s.get))
	group.POST("/invoices/:id/void", apihttp.ID(s.void))
}

func (s service) register(ctx context.Context, input registerInput) (store.ID, error) {
	if input.Direction != "issued" && input.Direction != "received" {
		return store.ID{}, store.Invalid("invoice direction must be issued or received")
	}
	result, err := store.Insert(ctx, s.db, `
		INSERT INTO invoice_register (party_id, direction, invoice_no, invoiced_on, currency,
			gross_amount_minor, tax_amount_minor, statement_id, notes)
		SELECT p.id, $2, $3, $4, $5, $6, $7, $8, nullif($9, '') FROM master_parties p
		WHERE p.id = $1 AND (($8::bigint IS NULL AND CASE $2 WHEN 'issued' THEN p.is_customer ELSE p.is_supplier END)
			OR EXISTS (SELECT 1 FROM settlement_statements s WHERE s.id = $8 AND s.party_id = p.id
				AND s.currency = $5 AND s.status = 'confirmed'
				AND s.direction = CASE $2 WHEN 'issued' THEN 'receivable' ELSE 'payable' END))
		RETURNING id`, input.PartyID, input.Direction, input.InvoiceNo, input.InvoicedOn,
		input.Currency, input.GrossAmountMinor, input.TaxAmountMinor, input.StatementID, input.Notes)
	return result, store.RequireReference(err, "invoice requires a matching party role or confirmed statement")
}

func (s service) list(ctx context.Context, after, limit int64) (json.RawMessage, error) {
	return store.JSON(ctx, s.db, `SELECT coalesce(jsonb_agg(to_jsonb(i) - 'statement_direction' ORDER BY i.id), '[]')
		FROM (SELECT * FROM invoice_register WHERE id > $1 ORDER BY id LIMIT $2) i`, after, limit)
}

func (s service) get(ctx context.Context, id int64) (json.RawMessage, error) {
	return store.JSON(ctx, s.db, `SELECT to_jsonb(i) - 'statement_direction' FROM invoice_register i WHERE id = $1`, id)
}

func (s service) void(ctx context.Context, id int64) (store.ID, error) {
	return store.Insert(ctx, s.db, `UPDATE invoice_register SET status = 'void' WHERE id = $1 RETURNING id`, id)
}
