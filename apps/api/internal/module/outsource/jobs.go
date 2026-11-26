package outsource

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/apihttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/module/production"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

type dispatchInput struct {
	JobNo         string  `json:"job_no"`
	WorkOrderID   int64   `json:"work_order_id"`
	SupplierID    int64   `json:"supplier_id"`
	OperationID   int64   `json:"operation_id"`
	QuantityMilli int64   `json:"quantity_milli"`
	Currency      string  `json:"currency"`
	AmountMinor   *int64  `json:"amount_minor"`
	SentOn        string  `json:"sent_on"`
	DueOn         *string `json:"due_on"`
}

type returnInput struct {
	ReturnNo      string `json:"return_no"`
	ReceivedOn    string `json:"received_on"`
	AcceptedMilli int64  `json:"accepted_milli"`
	RejectedMilli int64  `json:"rejected_milli"`
}

type service struct{ db *sql.DB }

func Bind(group *ghttp.RouterGroup, db *sql.DB) {
	s := service{db}
	group.GET("/outsource-jobs", apihttp.List(store.List(db, `SELECT j.*, (SELECT max(received_on) FROM outsource_returns WHERE job_id = j.id) AS last_received_on
		FROM outsource_jobs j WHERE id > $1 ORDER BY id LIMIT $2`)))
	group.POST("/outsource-jobs", apihttp.Body(s.dispatch))
	group.GET("/outsource-jobs/:id", apihttp.ID(s.get))
	group.POST("/outsource-jobs/:id/returns", apihttp.BodyID(s.receive))
}

func (s service) dispatch(ctx context.Context, input dispatchInput) (store.ID, error) {
	if input.AmountMinor == nil {
		return store.ID{}, store.Invalid("amount_minor is required")
	}
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		capacity, status, err := production.LockWorkOrder(ctx, tx, input.WorkOrderID)
		if err != nil {
			return store.ID{}, store.RequireReference(err, "a work order is required")
		}
		if !production.IsOpen(status) {
			return store.ID{}, store.Conflict("work order is not open")
		}
		if err := production.CheckCapacity(ctx, tx, input.WorkOrderID, input.OperationID, capacity, input.QuantityMilli); err != nil {
			return store.ID{}, err
		}
		job, err := store.Insert(ctx, tx, `
			INSERT INTO outsource_jobs (job_no, work_order_id, supplier_id, operation_name, operation_id,
				quantity_milli, currency, amount_minor, status, sent_on, due_on)
			SELECT $1, $2, p.id, o.name, o.id, $5, $6, $7, 'sent', $8, $9
			FROM master_parties p CROSS JOIN master_operations o
			WHERE p.id = $3 AND p.active AND p.is_supplier AND o.id = $4 AND o.active RETURNING id`,
			input.JobNo, input.WorkOrderID, input.SupplierID, input.OperationID, input.QuantityMilli,
			input.Currency, *input.AmountMinor, input.SentOn, input.DueOn)
		if err != nil {
			return store.ID{}, store.RequireReference(err, "an active supplier and operation are required")
		}
		return job, production.MarkInProgress(ctx, tx, input.WorkOrderID)
	})
}

func (s service) get(ctx context.Context, id int64) (json.RawMessage, error) {
	return store.JSON(ctx, s.db, `
		SELECT to_jsonb(j) || jsonb_build_object('returns', coalesce(
			(SELECT jsonb_agg(r ORDER BY r.id) FROM outsource_returns r WHERE r.job_id = j.id), '[]'))
		FROM outsource_jobs j WHERE j.id = $1`, id)
}

func (s service) receive(ctx context.Context, id int64, input returnInput) (store.ID, error) {
	if input.AcceptedMilli < 0 || input.RejectedMilli < 0 || (input.AcceptedMilli == 0 && input.RejectedMilli == 0) {
		return store.ID{}, store.Invalid("a return requires positive accepted or rejected quantity")
	}
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		var workOrderID int64
		if err := tx.QueryRowContext(ctx, "SELECT work_order_id FROM outsource_jobs WHERE id = $1", id).Scan(&workOrderID); err != nil {
			return store.ID{}, err
		}
		if _, _, err := production.LockWorkOrder(ctx, tx, workOrderID); err != nil {
			return store.ID{}, err
		}
		var status string
		var sent, returned int64
		var validDate bool
		err := tx.QueryRowContext(ctx, `
			SELECT status, quantity_milli, sent_on IS NULL OR $2::date >= sent_on
			FROM outsource_jobs WHERE id = $1 FOR NO KEY UPDATE`, id, input.ReceivedOn).Scan(&status, &sent, &validDate)
		if err != nil {
			return store.ID{}, err
		}
		if status != "sent" {
			return store.ID{}, store.Conflict("only a sent outsource job can receive returns")
		}
		if !validDate {
			return store.ID{}, store.Invalid("return date precedes dispatch")
		}
		err = tx.QueryRowContext(ctx, `
			SELECT coalesce(sum(accepted_milli::numeric + rejected_milli), 0)
			FROM outsource_returns WHERE job_id = $1`, id).Scan(&returned)
		if err != nil {
			return store.ID{}, err
		}
		remaining := sent - returned
		if input.AcceptedMilli > remaining || input.RejectedMilli > remaining-input.AcceptedMilli {
			return store.ID{}, store.Conflict("return quantity exceeds outstanding quantity")
		}
		result, err := store.Insert(ctx, tx, `
			INSERT INTO outsource_returns (return_no, job_id, received_on, accepted_milli, rejected_milli)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			input.ReturnNo, id, input.ReceivedOn, input.AcceptedMilli, input.RejectedMilli)
		if err != nil {
			return store.ID{}, err
		}
		if input.RejectedMilli == remaining-input.AcceptedMilli {
			_, err = tx.ExecContext(ctx, "UPDATE outsource_jobs SET status = 'closed' WHERE id = $1", id)
		}
		return result, err
	})
}

// PayableAmount uses the accepted share of the price. The carrying date of a
// fully returned job is the latest return date, regardless of entry order.
func PayableAmount(ctx context.Context, tx *sql.Tx, id, partyID int64, currency, start, end string) (amount int64, err error) {
	err = tx.QueryRowContext(ctx, `
		SELECT div(j.amount_minor::numeric * r.accepted, j.quantity_milli)
		FROM outsource_jobs j
		CROSS JOIN LATERAL (
			SELECT sum(accepted_milli) AS accepted,
				sum(accepted_milli::numeric + rejected_milli) AS returned, max(received_on) AS received_on
			FROM outsource_returns WHERE job_id = j.id
		) r
		WHERE j.id = $1 AND j.supplier_id = $2 AND j.currency = $3 AND j.status = 'closed'
			AND r.returned = j.quantity_milli AND r.received_on BETWEEN $4 AND $5`,
		id, partyID, currency, start, end).Scan(&amount)
	return
}
