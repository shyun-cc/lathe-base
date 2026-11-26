package sales

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/apihttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

type orderInput struct {
	OrderNo    string      `json:"order_no"`
	CustomerID int64       `json:"customer_id"`
	Currency   string      `json:"currency"`
	OrderedOn  string      `json:"ordered_on"`
	Lines      []lineInput `json:"lines"`
}

type lineInput struct {
	ProductID     int64   `json:"product_id"`
	QuantityMilli int64   `json:"quantity_milli"`
	AmountMinor   *int64  `json:"amount_minor"`
	DueOn         *string `json:"due_on"`
}

type service struct{ db *sql.DB }

func Bind(group *ghttp.RouterGroup, db *sql.DB) {
	s := service{db}
	group.POST("/sales-orders", apihttp.Body(s.create))
	group.GET("/sales-orders/:id", apihttp.ID(s.get))
	group.POST("/sales-orders/:id/confirm", apihttp.ID(s.confirm))
}

func (s service) create(ctx context.Context, input orderInput) (store.ID, error) {
	if len(input.Lines) == 0 || len(input.Lines) > 200 {
		return store.ID{}, store.Invalid("an order must contain 1 to 200 lines")
	}
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		// The INSERT takes a snapshot of the current customer and the line writes
		// snapshot the product name and unit for future reference.
		order, err := store.Insert(ctx, tx, `
			INSERT INTO sales_orders (order_no, customer_id, currency, ordered_on)
			SELECT $1, id, $3, $4 FROM master_parties WHERE id = $2 AND is_customer AND active
			RETURNING id`, input.OrderNo, input.CustomerID, input.Currency, input.OrderedOn)
		if err != nil {
			return store.ID{}, store.RequireReference(err, "an active customer is required")
		}
		for i, line := range input.Lines {
			if line.AmountMinor == nil {
				return store.ID{}, store.Invalid("amount_minor is required on each order line")
			}
			_, err := store.Insert(ctx, tx, `
				INSERT INTO sales_order_lines (order_id, line_no, product_id, unit_id, product_name,
					quantity_milli, amount_minor, due_on)
				SELECT $1, $2, p.id, p.unit_id, p.name, $4, $5, $6
				FROM master_products p JOIN master_units u ON u.id = p.unit_id
				WHERE p.id = $3 AND p.active AND u.active RETURNING id`,
				order.ID, i+1, line.ProductID, line.QuantityMilli, *line.AmountMinor, line.DueOn)
			if err != nil {
				return store.ID{}, store.RequireReference(err, "an active product and unit are required")
			}
		}
		return order, nil
	})
}

func (s service) get(ctx context.Context, id int64) (json.RawMessage, error) {
	return store.JSON(ctx, s.db, `
		SELECT to_jsonb(o) || jsonb_build_object('lines', coalesce(
			(SELECT jsonb_agg(l ORDER BY l.line_no) FROM sales_order_lines l WHERE l.order_id = o.id), '[]'))
		FROM sales_orders o WHERE o.id = $1`, id)
}

func (s service) confirm(ctx context.Context, id int64) (store.ID, error) {
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		var status string
		var hasLines bool
		err := tx.QueryRowContext(ctx, `
			SELECT status, EXISTS (SELECT 1 FROM sales_order_lines WHERE order_id = $1)
			FROM sales_orders WHERE id = $1 FOR NO KEY UPDATE`, id).Scan(&status, &hasLines)
		if err != nil {
			return store.ID{}, err
		}
		switch status {
		case "confirmed":
			return store.ID{ID: id}, nil
		case "draft":
			if hasLines {
				return store.Insert(ctx, tx, "UPDATE sales_orders SET status = 'confirmed' WHERE id = $1 RETURNING id", id)
			}
		}
		return store.ID{}, store.Conflict("only a draft order with lines can be confirmed")
	})
}
