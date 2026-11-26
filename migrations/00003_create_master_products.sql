-- +goose Up
CREATE TABLE master_products (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sku text NOT NULL UNIQUE CHECK (btrim(sku) <> ''),
    name text NOT NULL CHECK (btrim(name) <> ''),
    unit_id bigint NOT NULL REFERENCES master_units (id),
    active boolean NOT NULL DEFAULT true
);

CREATE INDEX master_products_unit_idx ON master_products (unit_id);

-- +goose Down
DROP TABLE master_products;
