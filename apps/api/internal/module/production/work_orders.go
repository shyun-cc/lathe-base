package production

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/apihttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

type batch struct {
	WorkOrderNo   string  `json:"work_order_no"`
	QuantityMilli int64   `json:"quantity_milli"`
	DueOn         *string `json:"due_on"`
}

type splitInput struct {
	Batches []batch `json:"batches"`
}

type reportInput struct {
	ReportNo      string `json:"report_no"`
	OperationID   int64  `json:"operation_id"`
	EmployeeID    int64  `json:"employee_id"`
	ReportedOn    string `json:"reported_on"`
	QuantityMilli int64  `json:"quantity_milli"`
}

type completionInput struct {
	OperationID int64 `json:"operation_id"`
}

type service struct{ db *sql.DB }

func Bind(group *ghttp.RouterGroup, db *sql.DB) {
	s := service{db}
	group.POST("/sales-order-lines/:id/work-orders", apihttp.BodyID(s.split))
	group.GET("/work-orders", apihttp.List(store.List(db, `SELECT * FROM production_work_orders WHERE id > $1 ORDER BY id LIMIT $2`)))
	group.GET("/work-orders/:id", apihttp.ID(s.get))
	group.POST("/work-orders/:id/reports", apihttp.BodyID(s.report))
	group.POST("/work-orders/:id/complete", apihttp.BodyID(s.complete))
}

func (s service) split(ctx context.Context, lineID int64, input splitInput) ([]store.ID, error) {
	if len(input.Batches) > 200 {
		return nil, store.Invalid("up to 200 batches may be submitted at a time")
	}
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) ([]store.ID, error) {
		line := OrderLine{ID: lineID}
		var status string
		err := tx.QueryRowContext(ctx, `
			SELECT l.quantity_milli, o.status FROM sales_order_lines l
			JOIN sales_orders o ON o.id = l.order_id
			WHERE l.id = $1 FOR NO KEY UPDATE OF l`, lineID).Scan(&line.Ordered, &status)
		if err != nil {
			return nil, err
		}
		if status != "confirmed" {
			return nil, store.Conflict("only a confirmed sales order can be split")
		}
		err = tx.QueryRowContext(ctx, `
			SELECT coalesce(sum(quantity_milli), 0) FROM production_work_orders
			WHERE sales_order_line_id = $1 AND status <> 'cancelled'`, lineID).Scan(&line.Assigned)
		if err != nil {
			return nil, err
		}
		quantities := make([]int64, len(input.Batches))
		for i, batch := range input.Batches {
			quantities[i] = batch.QuantityMilli
		}
		drafts, err := Split(line, quantities)
		if err != nil {
			return nil, store.Conflict(err.Error())
		}
		results := make([]store.ID, 0, len(drafts))
		for i, draft := range drafts {
			batch := input.Batches[i]
			workOrder, err := store.Insert(ctx, tx, `
				INSERT INTO production_work_orders (work_order_no, sales_order_line_id, quantity_milli, status, due_on)
				VALUES ($1, $2, $3, 'released', $4) RETURNING id`,
				batch.WorkOrderNo, draft.OrderLineID, draft.Quantity, batch.DueOn)
			if err != nil {
				return nil, err
			}
			results = append(results, workOrder)
		}
		return results, nil
	})
}

func (s service) get(ctx context.Context, id int64) (json.RawMessage, error) {
	return store.JSON(ctx, s.db, `
		SELECT to_jsonb(w) || jsonb_build_object('reports', coalesce(
			(SELECT jsonb_agg(r ORDER BY r.id) FROM production_reports r WHERE r.work_order_id = w.id), '[]'))
		FROM production_work_orders w WHERE w.id = $1`, id)
}

// LockWorkOrder serializes reports, dispatches, returns and completion on a work order.
func LockWorkOrder(ctx context.Context, tx *sql.Tx, id int64) (quantity int64, status string, err error) {
	err = tx.QueryRowContext(ctx, `
		SELECT quantity_milli, status FROM production_work_orders WHERE id = $1 FOR NO KEY UPDATE`, id).
		Scan(&quantity, &status)
	return
}

func IsOpen(status string) bool {
	return status == "released" || status == "in_progress"
}

// CheckCapacity expects the caller to hold the work order lock. A rejected return
// frees capacity for rework. Parts still at the supplier remain reserved.
func CheckCapacity(ctx context.Context, tx *sql.Tx, id, operation, capacity, quantity int64) error {
	if quantity <= 0 {
		return store.Invalid("quantity_milli must be positive")
	}
	var fits bool
	err := tx.QueryRowContext(ctx, `
		SELECT $3::numeric <= $4::numeric
			- coalesce((SELECT sum(quantity_milli) FROM production_reports
				WHERE work_order_id = $1 AND operation_id = $2), 0)
			- coalesce((SELECT sum(j.quantity_milli - coalesce(
				(SELECT sum(r.rejected_milli) FROM outsource_returns r WHERE r.job_id = j.id), 0))
				FROM outsource_jobs j WHERE j.work_order_id = $1 AND j.operation_id = $2
					AND j.status IN ('sent', 'closed')), 0)`, id, operation, quantity, capacity).Scan(&fits)
	if err != nil {
		return err
	}
	if !fits {
		return store.Conflict("quantity exceeds remaining capacity for the work order operation")
	}
	return nil
}

func MarkInProgress(ctx context.Context, tx *sql.Tx, id int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE production_work_orders SET status = 'in_progress' WHERE id = $1 AND status = 'released'`, id)
	return err
}

func (s service) report(ctx context.Context, id int64, input reportInput) (store.ID, error) {
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		capacity, status, err := LockWorkOrder(ctx, tx, id)
		if err != nil {
			return store.ID{}, err
		}
		if !IsOpen(status) {
			return store.ID{}, store.Conflict("work order is not open")
		}
		if err := CheckCapacity(ctx, tx, id, input.OperationID, capacity, input.QuantityMilli); err != nil {
			return store.ID{}, err
		}
		report, err := store.Insert(ctx, tx, `
			INSERT INTO production_reports (report_no, work_order_id, operation_id, employee_id,
				operation_name, reported_on, quantity_milli)
			SELECT $1, $2, o.id, e.id, o.name, $5, $6 FROM master_operations o
			CROSS JOIN master_employees e WHERE o.id = $3 AND o.active AND e.id = $4 AND e.active RETURNING id`,
			input.ReportNo, id, input.OperationID, input.EmployeeID, input.ReportedOn, input.QuantityMilli)
		if err != nil {
			return store.ID{}, store.RequireReference(err, "an active operation and employee are required")
		}
		return report, MarkInProgress(ctx, tx, id)
	})
}

func (s service) complete(ctx context.Context, id int64, input completionInput) (store.ID, error) {
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		capacity, status, err := LockWorkOrder(ctx, tx, id)
		if err != nil {
			return store.ID{}, err
		}
		if status == "completed" {
			return store.ID{ID: id}, nil
		}
		if !IsOpen(status) || input.OperationID <= 0 {
			return store.ID{}, store.Conflict("completion requires an open work order and a final operation")
		}
		var finished bool
		err = tx.QueryRowContext(ctx, `
			SELECT NOT EXISTS (SELECT 1 FROM outsource_jobs WHERE work_order_id = $1 AND status IN ('sent', 'draft'))
				AND $3::numeric = coalesce((SELECT sum(quantity_milli) FROM production_reports
					WHERE work_order_id = $1 AND operation_id = $2), 0)
				+ coalesce((SELECT sum(r.accepted_milli) FROM outsource_returns r
					JOIN outsource_jobs j ON j.id = r.job_id
					WHERE j.work_order_id = $1 AND j.operation_id = $2 AND j.status = 'closed'), 0)`,
			id, input.OperationID, capacity).Scan(&finished)
		if err != nil {
			return store.ID{}, err
		}
		if !finished {
			return store.ID{}, store.Conflict("the final operation needs the full quantity and all outsource jobs must be resolved")
		}
		return store.Insert(ctx, tx, `UPDATE production_work_orders SET status = 'completed' WHERE id = $1 RETURNING id`, id)
	})
}
