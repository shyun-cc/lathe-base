package shipping

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/shyun-cc/lathe-base/apps/api/internal/module/printing"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

func PrintProvider(db *sql.DB) printing.Provider {
	return printing.Provider{
		Definition: printing.Standard("送货单",
			[]printing.Field{
				{Key: "number", Label: "送货单号"},
				{Key: "party", Label: "客户"},
				{Key: "order", Label: "销售订单"},
				{Key: "date", Label: "发货日期"},
			},
			[]printing.Field{
				{Key: "product", Label: "物料"},
				{Key: "unit", Label: "单位"},
				{Key: "quantity", Label: "发货数量"},
			},
			[]printing.Cell{{Label: "发货人", Field: "", Span: 1}, {Label: "收货签字", Field: "", Span: 1}},
		),
		Load: func(ctx context.Context, id int64) (json.RawMessage, error) {
			return store.JSON(ctx, db, `
				SELECT jsonb_build_object('fields', jsonb_build_object(
	'number', s.shipment_no, 'party', p.name, 'order', o.order_no, 'date', s.dispatched_on),
	'rows', coalesce(jsonb_agg(jsonb_build_object('product', l.product_name, 'unit', u.name,
	'quantity', trim_scale(d.quantity_milli::numeric / 1000)::text) ORDER BY l.line_no)
	FILTER (WHERE d.sales_order_line_id IS NOT NULL), '[]'))
	FROM shipping_shipments s JOIN sales_orders o ON o.id = s.order_id
	JOIN master_parties p ON p.id = s.customer_id
	LEFT JOIN shipping_shipment_lines d ON d.shipment_id = s.id
	LEFT JOIN sales_order_lines l ON l.id = d.sales_order_line_id
	LEFT JOIN master_units u ON u.id = l.unit_id
	WHERE s.id = $1 GROUP BY s.id, p.name, o.order_no`, id)
		},
	}
}
