-- +goose Up
CREATE TABLE settlement_allocations (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_key text NOT NULL UNIQUE CHECK (btrim(request_key) <> ''),
    statement_id bigint NOT NULL,
    cash_entry_id bigint NOT NULL,
    party_id bigint NOT NULL,
    direction text NOT NULL,
    currency text NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (statement_id, party_id, direction, currency)
        REFERENCES settlement_statements (id, party_id, direction, currency),
    FOREIGN KEY (cash_entry_id, party_id, direction, currency)
        REFERENCES settlement_cash_entries (id, party_id, direction, currency)
);

CREATE INDEX settlement_allocations_statement_idx ON settlement_allocations (statement_id);
CREATE INDEX settlement_allocations_cash_entry_idx ON settlement_allocations (cash_entry_id);

-- +goose Down
DROP TABLE settlement_allocations;
