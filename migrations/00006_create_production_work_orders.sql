-- +goose Up
CREATE TABLE production_work_orders (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    work_order_no text NOT NULL UNIQUE CHECK (btrim(work_order_no) <> ''),
    sales_order_line_id bigint NOT NULL REFERENCES sales_order_lines (id),
    quantity_milli bigint NOT NULL CHECK (quantity_milli > 0),
    status text NOT NULL DEFAULT 'planned'
        CHECK (status IN ('planned', 'released', 'in_progress', 'completed', 'cancelled')),
    due_on date,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX production_work_orders_sales_line_idx ON production_work_orders (sales_order_line_id);

-- +goose Down
DROP TABLE production_work_orders;
