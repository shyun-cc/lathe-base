-- +goose Up
ALTER TABLE shipping_shipment_lines ADD COLUMN amount_minor bigint CHECK (amount_minor >= 0);

-- +goose Down
ALTER TABLE shipping_shipment_lines DROP COLUMN amount_minor;
