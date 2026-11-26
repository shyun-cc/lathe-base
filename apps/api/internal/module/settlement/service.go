package settlement

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/apihttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/outsource"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/shipping"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

type accountInput struct {
	PartyID   int64     `json:"party_id"`
	Currency  string    `json:"currency"`
	Direction Direction `json:"direction"`
}

func (a accountInput) validate() error {
	if a.PartyID <= 0 || !isCurrencyCode(a.Currency) ||
		(a.Direction != Receivable && a.Direction != Payable) {
		return store.Invalid("a party, uppercase currency code and receivable or payable direction are required")
	}
	return nil
}

type statementInput struct {
	accountInput
	StatementNo string  `json:"statement_no"`
	PeriodStart string  `json:"period_start"`
	PeriodEnd   string  `json:"period_end"`
	SourceIDs   []int64 `json:"source_ids"`
}

type cashInput struct {
	accountInput
	EntryNo     string `json:"entry_no"`
	BookedOn    string `json:"booked_on"`
	AmountMinor int64  `json:"amount_minor"`
	Method      string `json:"method"`
	Reference   string `json:"reference"`
}

type allocationInput struct {
	RequestKey  string `json:"request_key"`
	StatementID int64  `json:"statement_id"`
	CashEntryID int64  `json:"cash_entry_id"`
	AmountMinor int64  `json:"amount_minor"`
}

type service struct{ db *sql.DB }

func Bind(group *ghttp.RouterGroup, db *sql.DB) {
	s := service{db}
	group.GET("/statements", apihttp.List(store.List(db, `SELECT * FROM settlement_statements WHERE id > $1 ORDER BY id LIMIT $2`)))
	group.POST("/statements", apihttp.Body(s.createStatement))
	group.GET("/statements/:id", apihttp.ID(s.statement))
	group.GET("/cash-entries", apihttp.List(store.List(db, `SELECT c.*, c.amount_minor - coalesce(
		(SELECT sum(a.amount_minor) FROM settlement_allocations a WHERE a.cash_entry_id = c.id), 0) AS remaining_minor
		FROM settlement_cash_entries c WHERE id > $1 ORDER BY id LIMIT $2`)))
	group.POST("/cash-entries", apihttp.Body(s.postCash))
	group.GET("/cash-entries/:id", apihttp.ID(s.cash))
	group.POST("/allocations", apihttp.Body(s.allocate))
}

func (s service) createStatement(ctx context.Context, input statementInput) (store.ID, error) {
	if err := input.validate(); err != nil {
		return store.ID{}, err
	}
	if len(input.SourceIDs) == 0 || len(input.SourceIDs) > 200 {
		return store.ID{}, store.Invalid("a statement must contain 1 to 200 source documents")
	}
	slices.Sort(input.SourceIDs)
	for i, id := range input.SourceIDs {
		if id <= 0 || (i > 0 && id == input.SourceIDs[i-1]) {
			return store.ID{}, store.Invalid("source_ids must contain distinct positive identifiers")
		}
	}

	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		statement, err := store.Insert(ctx, tx, `
			INSERT INTO settlement_statements
				(statement_no, party_id, direction, currency, period_start, period_end, amount_minor)
			VALUES ($1, $2, $3, $4, $5, $6, 0) RETURNING id`, input.StatementNo, input.PartyID,
			input.Direction, input.Currency, input.PeriodStart, input.PeriodEnd)
		if err != nil {
			return store.ID{}, err
		}
		var total int64
		for _, id := range input.SourceIDs {
			var shipmentID, jobID *int64
			sourceAmount := shipping.ReceivableAmount
			if input.Direction == Receivable {
				shipmentID = &id
			} else {
				jobID = &id
				sourceAmount = outsource.PayableAmount
			}
			amount, err := sourceAmount(ctx, tx, id, input.PartyID, input.Currency, input.PeriodStart, input.PeriodEnd)
			if err != nil {
				return store.ID{}, store.RequireReference(err, "source must be confirmed and match the party, currency and period")
			}
			if amount > math.MaxInt64-total {
				return store.ID{}, store.Invalid("statement total exceeds the supported amount")
			}
			total += amount
			_, err = tx.ExecContext(ctx, `
				INSERT INTO settlement_statement_lines
					(statement_id, party_id, direction, currency, shipment_id, outsource_job_id, amount_minor)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`, statement.ID, input.PartyID,
				input.Direction, input.Currency, shipmentID, jobID, amount)
			if err != nil {
				return store.ID{}, err
			}
		}
		return store.Insert(ctx, tx, `
			UPDATE settlement_statements SET amount_minor = $2, status = 'confirmed'
			WHERE id = $1 RETURNING id`, statement.ID, total)
	})
}

func (s service) statement(ctx context.Context, id int64) (json.RawMessage, error) {
	return store.JSON(ctx, s.db, `
		SELECT to_jsonb(s) || jsonb_build_object('lines', coalesce(
			(SELECT jsonb_agg(l ORDER BY l.id) FROM settlement_statement_lines l WHERE l.statement_id = s.id), '[]'),
			'allocations', coalesce((SELECT jsonb_agg(jsonb_build_object(
				'id', a.id, 'statement_id', a.statement_id, 'cash_entry_id', a.cash_entry_id, 'amount_minor', a.amount_minor)
				ORDER BY a.id) FROM settlement_allocations a WHERE a.statement_id = s.id), '[]'),
			'remaining_minor', s.amount_minor - coalesce(
				(SELECT sum(a.amount_minor) FROM settlement_allocations a WHERE a.statement_id = s.id), 0))
		FROM settlement_statements s WHERE s.id = $1`, id)
}

func (s service) postCash(ctx context.Context, input cashInput) (store.ID, error) {
	if err := input.validate(); err != nil {
		return store.ID{}, err
	}
	result, err := store.Insert(ctx, s.db, `
		INSERT INTO settlement_cash_entries (entry_no, party_id, direction, currency, booked_on,
			amount_minor, method, reference)
		SELECT $1, p.id, $3, $4, $5, $6, $7, nullif($8, '') FROM master_parties p
		WHERE p.id = $2 AND (CASE $3 WHEN 'receivable' THEN p.is_customer ELSE p.is_supplier END
			OR EXISTS (SELECT 1 FROM settlement_statements s WHERE s.party_id = p.id
				AND s.direction = $3 AND s.currency = $4 AND s.status = 'confirmed'))
		RETURNING id`, input.EntryNo, input.PartyID, input.Direction, input.Currency,
		input.BookedOn, input.AmountMinor, input.Method, input.Reference)
	return result, store.RequireReference(err, "party has no matching role or confirmed statement")
}

func (s service) cash(ctx context.Context, id int64) (json.RawMessage, error) {
	return store.JSON(ctx, s.db, `
		SELECT to_jsonb(c) || jsonb_build_object('allocations', coalesce((SELECT jsonb_agg(jsonb_build_object(
				'id', a.id, 'statement_id', a.statement_id, 'cash_entry_id', a.cash_entry_id, 'amount_minor', a.amount_minor)
				ORDER BY a.id) FROM settlement_allocations a WHERE a.cash_entry_id = c.id), '[]'),
			'remaining_minor', c.amount_minor - coalesce(
			(SELECT sum(a.amount_minor) FROM settlement_allocations a WHERE a.cash_entry_id = c.id), 0))
		FROM settlement_cash_entries c WHERE c.id = $1`, id)
}

func (s service) allocate(ctx context.Context, input allocationInput) (store.ID, error) {
	input.RequestKey = strings.TrimSpace(input.RequestKey)
	if input.RequestKey == "" || len(input.RequestKey) > 200 || input.StatementID <= 0 ||
		input.CashEntryID <= 0 || input.AmountMinor <= 0 {
		return store.ID{}, store.Invalid("request_key, statement_id, cash_entry_id and a positive amount_minor are required")
	}
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		// Serialize the key even if two requests refer to different accounts.
		// A hash collision can delay another request but never aliases its key.
		_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", input.RequestKey)
		if err != nil {
			return store.ID{}, err
		}
		var prior store.ID
		var statementID, cashID, amount int64
		err = tx.QueryRowContext(ctx, `
			SELECT id, statement_id, cash_entry_id, amount_minor FROM settlement_allocations WHERE request_key = $1`,
			input.RequestKey).Scan(&prior.ID, &statementID, &cashID, &amount)
		if err == nil {
			if statementID != input.StatementID || cashID != input.CashEntryID || amount != input.AmountMinor {
				return store.ID{}, store.Conflict("request_key already belongs to a different allocation")
			}
			return prior, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return store.ID{}, err
		}

		var statement, cash Entry
		var statementStatus, cashStatus string
		err = tx.QueryRowContext(ctx, `
			SELECT party_id, currency, direction, amount_minor, status FROM settlement_statements
			WHERE id = $1 FOR NO KEY UPDATE`, input.StatementID).Scan(
			&statement.PartyID, &statement.Currency, &statement.Direction, &statement.Remaining, &statementStatus)
		if err != nil {
			return store.ID{}, err
		}
		err = tx.QueryRowContext(ctx, `
			SELECT party_id, currency, direction, amount_minor, status FROM settlement_cash_entries
			WHERE id = $1 FOR NO KEY UPDATE`, input.CashEntryID).Scan(
			&cash.PartyID, &cash.Currency, &cash.Direction, &cash.Remaining, &cashStatus)
		if err != nil {
			return store.ID{}, err
		}
		if statementStatus != "confirmed" || cashStatus != "posted" {
			return store.ID{}, store.Conflict("only confirmed statements and posted cash entries can be allocated")
		}
		err = tx.QueryRowContext(ctx, `
			SELECT $3::numeric - coalesce((SELECT sum(amount_minor) FROM settlement_allocations WHERE statement_id = $1), 0),
				$4::numeric - coalesce((SELECT sum(amount_minor) FROM settlement_allocations WHERE cash_entry_id = $2), 0)`,
			input.StatementID, input.CashEntryID, statement.Remaining, cash.Remaining).
			Scan(&statement.Remaining, &cash.Remaining)
		if err != nil {
			return store.ID{}, err
		}
		if _, err := Allocate(statement, cash, input.AmountMinor); err != nil {
			return store.ID{}, store.Conflict(err.Error())
		}
		return store.Insert(ctx, tx, `
			INSERT INTO settlement_allocations (request_key, statement_id, cash_entry_id, party_id, direction, currency, amount_minor)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, input.RequestKey, input.StatementID,
			input.CashEntryID, statement.PartyID, statement.Direction, statement.Currency, input.AmountMinor)
	})
}
