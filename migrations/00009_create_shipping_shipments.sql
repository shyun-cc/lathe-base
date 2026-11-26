-- +goose Up
CREATE TABLE shipping_shipments (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shipment_no text NOT NULL UNIQUE CHECK (btrim(shipment_no) <> ''),
    order_id bigint NOT NULL,
    customer_id bigint NOT NULL,
    currency text NOT NULL,
    status text NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'dispatched', 'delivered', 'cancelled')),
    dispatched_on date,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (order_id, customer_id, currency) REFERENCES sales_orders (id, customer_id, currency),
    UNIQUE (id, customer_id, currency),
    UNIQUE (id, order_id)
);

CREATE INDEX shipping_shipments_order_idx ON shipping_shipments (order_id);
CREATE INDEX shipping_shipments_customer_idx ON shipping_shipments (customer_id, dispatched_on);

-- +goose Down
DROP TABLE shipping_shipments;
