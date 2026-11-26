package shipping

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/apihttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

type shipmentInput struct {
	ShipmentNo string      `json:"shipment_no"`
	OrderID    int64       `json:"order_id"`
	Lines      []lineInput `json:"lines"`
}

type lineInput struct {
	SalesOrderLineID int64 `json:"sales_order_line_id"`
	QuantityMilli    int64 `json:"quantity_milli"`
}

type confirmInput struct {
	DispatchedOn string `json:"dispatched_on"`
}

type service struct{ db *sql.DB }

func Bind(group *ghttp.RouterGroup, db *sql.DB) {
	s := service{db}
	group.GET("/shipments", apihttp.List(store.List(db, `SELECT * FROM shipping_shipments WHERE id > $1 ORDER BY id LIMIT $2`)))
	group.POST("/shipments", apihttp.Body(s.create))
	group.GET("/shipments/:id", apihttp.ID(s.get))
	group.POST("/shipments/:id/confirm", apihttp.BodyID(s.confirm))
}

func (s service) create(ctx context.Context, input shipmentInput) (store.ID, error) {
	if len(input.Lines) == 0 || len(input.Lines) > 200 {
		return store.ID{}, store.Invalid("a shipment must contain 1 to 200 lines")
	}
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		shipment, err := store.Insert(ctx, tx, `
			INSERT INTO shipping_shipments (shipment_no, order_id, customer_id, currency)
			SELECT $1, id, customer_id, currency FROM sales_orders
			WHERE id = $2 AND status = 'confirmed' RETURNING id`, input.ShipmentNo, input.OrderID)
		if err != nil {
			return store.ID{}, store.RequireReference(err, "a confirmed sales order is required")
		}
		for _, line := range input.Lines {
			var saved int64
			err := tx.QueryRowContext(ctx, `
				INSERT INTO shipping_shipment_lines (shipment_id, order_id, sales_order_line_id, quantity_milli)
				SELECT $1, order_id, id, $4 FROM sales_order_lines
				WHERE order_id = $2 AND id = $3 AND $4 <= quantity_milli RETURNING sales_order_line_id`,
				shipment.ID, input.OrderID, line.SalesOrderLineID, line.QuantityMilli).Scan(&saved)
			if err != nil {
				return store.ID{}, store.RequireReference(err, "shipment line must belong to the sales order and fit its ordered quantity")
			}
		}
		return shipment, nil
	})
}

func (s service) get(ctx context.Context, id int64) (json.RawMessage, error) {
	return store.JSON(ctx, s.db, `
		SELECT to_jsonb(s) || jsonb_build_object('lines', coalesce(
			(SELECT jsonb_agg(to_jsonb(l) || jsonb_build_object('product_name', ol.product_name) ORDER BY ol.line_no)
				FROM shipping_shipment_lines l JOIN sales_order_lines ol ON ol.id = l.sales_order_line_id
				WHERE l.shipment_id = s.id), '[]'))
		FROM shipping_shipments s WHERE s.id = $1`, id)
}

func (s service) confirm(ctx context.Context, id int64, input confirmInput) (store.ID, error) {
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		// One order lock protects all its cumulative quantities. The subsequent
		// statements see shipments confirmed by the previous holder of the lock.
		var orderStatus string
		err := tx.QueryRowContext(ctx, `
			SELECT o.status FROM sales_orders o JOIN shipping_shipments s ON o.id = s.order_id
			WHERE s.id = $1 FOR NO KEY UPDATE OF o`, id).Scan(&orderStatus)
		if err != nil {
			return store.ID{}, err
		}
		var status string
		var sameDate bool
		err = tx.QueryRowContext(ctx, `
			SELECT status, dispatched_on IS NOT DISTINCT FROM $2::date
			FROM shipping_shipments WHERE id = $1 FOR NO KEY UPDATE`, id, input.DispatchedOn).Scan(&status, &sameDate)
		if err != nil {
			return store.ID{}, err
		}
		if (status == "dispatched" || status == "delivered") && sameDate {
			return store.ID{ID: id}, nil
		}
		if status != "draft" || orderStatus != "confirmed" {
			return store.ID{}, store.Conflict("only a draft shipment of a confirmed sales order can be confirmed")
		}
		var total, priced int64
		err = tx.QueryRowContext(ctx, `
			WITH quantities AS (
				SELECT line.sales_order_line_id, line.quantity_milli AS current_qty,
					sale.quantity_milli AS ordered_qty, sale.amount_minor,
					coalesce((SELECT sum(prev.quantity_milli)
						FROM shipping_shipment_lines prev JOIN shipping_shipments s ON s.id = prev.shipment_id
						WHERE prev.order_id = line.order_id AND prev.sales_order_line_id = line.sales_order_line_id
							AND s.status IN ('dispatched', 'delivered')), 0) AS previous_qty
				FROM shipping_shipment_lines line
				JOIN sales_order_lines sale ON sale.id = line.sales_order_line_id
				WHERE line.shipment_id = $1
			), priced AS (
				UPDATE shipping_shipment_lines line SET amount_minor =
					div(q.amount_minor * (q.previous_qty + q.current_qty), q.ordered_qty)
					- div(q.amount_minor * q.previous_qty, q.ordered_qty)
				FROM quantities q WHERE line.shipment_id = $1 AND line.sales_order_line_id = q.sales_order_line_id
					AND q.current_qty + q.previous_qty <= q.ordered_qty
				RETURNING 1
			)
			SELECT (SELECT count(*) FROM quantities), (SELECT count(*) FROM priced)`, id).Scan(&total, &priced)
		if err != nil {
			return store.ID{}, err
		}
		if total == 0 || total != priced {
			return store.ID{}, store.Conflict("shipment needs lines within the remaining ordered quantities")
		}
		return store.Insert(ctx, tx, `
			UPDATE shipping_shipments SET status = 'dispatched', dispatched_on = $2
			WHERE id = $1 RETURNING id`, id, input.DispatchedOn)
	})
}

// ReceivableAmount reads the amounts fixed at shipment confirmation. For
// shipments that predate pricing, apportion their share by shipment ID.
func ReceivableAmount(ctx context.Context, tx *sql.Tx, id, partyID int64, currency, start, end string) (amount int64, err error) {
	err = tx.QueryRowContext(ctx, `
		SELECT sum(coalesce(l.amount_minor,
			div(o.amount_minor * (prev.quantity + l.quantity_milli), o.quantity_milli)
			- div(o.amount_minor * prev.quantity, o.quantity_milli)))::bigint
		FROM shipping_shipments s JOIN shipping_shipment_lines l ON l.shipment_id = s.id
		JOIN sales_order_lines o ON o.id = l.sales_order_line_id
		LEFT JOIN LATERAL (
			SELECT coalesce(sum(p.quantity_milli), 0) AS quantity
			FROM shipping_shipment_lines p JOIN shipping_shipments history ON history.id = p.shipment_id
			WHERE l.amount_minor IS NULL AND p.amount_minor IS NULL AND p.order_id = l.order_id
				AND p.sales_order_line_id = l.sales_order_line_id AND p.shipment_id < l.shipment_id
				AND history.status IN ('dispatched', 'delivered')
		) prev ON true
		WHERE s.id = $1 AND s.customer_id = $2 AND s.currency = $3 AND s.status IN ('dispatched', 'delivered')
			AND s.dispatched_on BETWEEN $4 AND $5
		GROUP BY s.id`, id, partyID, currency, start, end).Scan(&amount)
	return
}
