-- +goose Up
CREATE TABLE sales_orders (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_no text NOT NULL UNIQUE CHECK (btrim(order_no) <> ''),
    customer_id bigint NOT NULL REFERENCES master_parties (id),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    ordered_on date NOT NULL,
    status text NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'confirmed', 'closed', 'cancelled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, customer_id, currency)
);

CREATE INDEX sales_orders_customer_date_idx ON sales_orders (customer_id, ordered_on);

-- +goose Down
DROP TABLE sales_orders;
