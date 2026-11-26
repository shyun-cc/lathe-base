-- +goose Up
CREATE TABLE shipping_shipment_lines (
    shipment_id bigint NOT NULL,
    order_id bigint NOT NULL,
    sales_order_line_id bigint NOT NULL,
    quantity_milli bigint NOT NULL CHECK (quantity_milli > 0),
    PRIMARY KEY (shipment_id, sales_order_line_id),
    FOREIGN KEY (shipment_id, order_id) REFERENCES shipping_shipments (id, order_id),
    FOREIGN KEY (order_id, sales_order_line_id) REFERENCES sales_order_lines (order_id, id)
);

CREATE INDEX shipping_shipment_lines_order_line_idx ON shipping_shipment_lines (order_id, sales_order_line_id);

-- +goose Down
DROP TABLE shipping_shipment_lines;
