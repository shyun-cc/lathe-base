package production

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/shyun-cc/lathe-base/apps/api/internal/module/printing"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

func PrintProvider(db *sql.DB) printing.Provider {
	return printing.Provider{
		Definition: printing.Standard("生产工单",
			[]printing.Field{
				{Key: "number", Label: "工单编号"},
				{Key: "order", Label: "销售订单"},
				{Key: "date", Label: "交期"},
			},
			[]printing.Field{
				{Key: "product", Label: "物料"},
				{Key: "unit", Label: "单位"},
				{Key: "quantity", Label: "生产数量"},
			},
			[]printing.Cell{{Label: "班组", Field: "", Span: 1}, {Label: "负责人", Field: "", Span: 1}},
		),
		Load: func(ctx context.Context, id int64) (json.RawMessage, error) {
			return store.JSON(ctx, db, `
				SELECT jsonb_build_object('fields', jsonb_build_object(
	'number', w.work_order_no, 'order', o.order_no, 'date', w.due_on),
	'rows', jsonb_build_array(jsonb_build_object('product', l.product_name, 'unit', u.name,
	'quantity', trim_scale(w.quantity_milli::numeric / 1000)::text)))
	FROM production_work_orders w JOIN sales_order_lines l ON l.id = w.sales_order_line_id
	JOIN sales_orders o ON o.id = l.order_id JOIN master_units u ON u.id = l.unit_id
	WHERE w.id = $1`, id)
		},
	}
}
