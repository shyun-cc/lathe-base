-- +goose Up
CREATE TABLE settlement_cash_entries (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    entry_no text NOT NULL UNIQUE CHECK (btrim(entry_no) <> ''),
    party_id bigint NOT NULL REFERENCES master_parties (id),
    direction text NOT NULL CHECK (direction IN ('receivable', 'payable')),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    booked_on date NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    method text NOT NULL CHECK (btrim(method) <> ''),
    reference text,
    status text NOT NULL DEFAULT 'posted' CHECK (status IN ('posted', 'void')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, party_id, direction, currency)
);

CREATE INDEX settlement_cash_entries_party_date_idx ON settlement_cash_entries (party_id, booked_on);

-- +goose Down
DROP TABLE settlement_cash_entries;
