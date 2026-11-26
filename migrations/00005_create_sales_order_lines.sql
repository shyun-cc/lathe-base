-- +goose Up
CREATE TABLE sales_order_lines (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id bigint NOT NULL REFERENCES sales_orders (id),
    line_no integer NOT NULL CHECK (line_no > 0),
    product_id bigint NOT NULL REFERENCES master_products (id),
    unit_id bigint NOT NULL REFERENCES master_units (id),
    product_name text NOT NULL CHECK (btrim(product_name) <> ''),
    quantity_milli bigint NOT NULL CHECK (quantity_milli > 0),
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    due_on date,
    UNIQUE (order_id, line_no),
    UNIQUE (order_id, id)
);

CREATE INDEX sales_order_lines_product_idx ON sales_order_lines (product_id);
CREATE INDEX sales_order_lines_unit_idx ON sales_order_lines (unit_id);

-- +goose Down
DROP TABLE sales_order_lines;
