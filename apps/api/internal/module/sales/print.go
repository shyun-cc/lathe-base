package sales

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/shyun-cc/lathe-base/apps/api/internal/module/printing"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

func PrintProvider(db *sql.DB) printing.Provider {
	return printing.Provider{
		Definition: printing.Standard("销售单",
			[]printing.Field{
				{Key: "number", Label: "订单编号"},
				{Key: "party", Label: "客户"},
				{Key: "date", Label: "下单日期"},
				{Key: "currency", Label: "币种"},
				{Key: "total", Label: "合计（最小单位）"},
			},
			[]printing.Field{
				{Key: "product", Label: "物料"},
				{Key: "unit", Label: "单位"},
				{Key: "quantity", Label: "数量"},
				{Key: "amount", Label: "金额（最小单位）"},
				{Key: "due", Label: "交期"},
			},
			[]printing.Cell{{Label: "合计（最小单位）", Field: "total", Span: 1}, {Label: "经办人", Field: "", Span: 1}},
		),
		Load: func(ctx context.Context, id int64) (json.RawMessage, error) {
			return store.JSON(ctx, db, `
				SELECT jsonb_build_object('fields', jsonb_build_object(
	'number', o.order_no, 'party', p.name, 'date', o.ordered_on, 'currency', o.currency,
	'total', coalesce(sum(l.amount_minor), 0)::text),
	'rows', coalesce(jsonb_agg(jsonb_build_object('product', l.product_name, 'unit', u.name,
	'quantity', trim_scale(l.quantity_milli::numeric / 1000)::text, 'amount', l.amount_minor::text,
	'due', l.due_on) ORDER BY l.line_no) FILTER (WHERE l.id IS NOT NULL), '[]'))
	FROM sales_orders o JOIN master_parties p ON p.id = o.customer_id
	LEFT JOIN sales_order_lines l ON l.order_id = o.id LEFT JOIN master_units u ON u.id = l.unit_id
	WHERE o.id = $1 GROUP BY o.id, p.name`, id)
		},
	}
}
