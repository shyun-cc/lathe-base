-- +goose Up
CREATE TABLE master_parties (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code text NOT NULL UNIQUE CHECK (btrim(code) <> ''),
    name text NOT NULL CHECK (btrim(name) <> ''),
    is_customer boolean NOT NULL DEFAULT false,
    is_supplier boolean NOT NULL DEFAULT false,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (is_customer OR is_supplier)
);

-- +goose Down
DROP TABLE master_parties;
