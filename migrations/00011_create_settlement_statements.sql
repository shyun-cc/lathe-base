-- +goose Up
CREATE TABLE settlement_statements (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    statement_no text NOT NULL UNIQUE CHECK (btrim(statement_no) <> ''),
    party_id bigint NOT NULL REFERENCES master_parties (id),
    direction text NOT NULL CHECK (direction IN ('receivable', 'payable')),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    period_start date NOT NULL,
    period_end date NOT NULL CHECK (period_end >= period_start),
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    status text NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'confirmed', 'void')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, party_id, direction, currency)
);

CREATE INDEX settlement_statements_party_period_idx ON settlement_statements (party_id, period_end);

-- +goose Down
DROP TABLE settlement_statements;
