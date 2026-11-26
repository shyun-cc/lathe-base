-- +goose Up
CREATE TABLE invoice_register (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    party_id bigint NOT NULL REFERENCES master_parties (id),
    direction text NOT NULL CHECK (direction IN ('issued', 'received')),
    invoice_no text NOT NULL CHECK (btrim(invoice_no) <> ''),
    invoiced_on date NOT NULL,
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    gross_amount_minor bigint NOT NULL CHECK (gross_amount_minor > 0),
    tax_amount_minor bigint NOT NULL CHECK (tax_amount_minor BETWEEN 0 AND gross_amount_minor),
    status text NOT NULL DEFAULT 'registered' CHECK (status IN ('registered', 'void')),
    statement_id bigint,
    statement_direction text GENERATED ALWAYS AS (
        CASE direction WHEN 'issued' THEN 'receivable' WHEN 'received' THEN 'payable' END
    ) STORED,
    notes text,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (statement_id, party_id, statement_direction, currency)
        REFERENCES settlement_statements (id, party_id, direction, currency)
);

CREATE UNIQUE INDEX invoice_register_issued_no_idx ON invoice_register (invoice_no) WHERE direction = 'issued';
CREATE UNIQUE INDEX invoice_register_received_no_idx ON invoice_register (party_id, invoice_no) WHERE direction = 'received';
CREATE INDEX invoice_register_party_date_idx ON invoice_register (party_id, invoiced_on);
CREATE INDEX invoice_register_statement_idx ON invoice_register (statement_id);

-- +goose Down
DROP TABLE invoice_register;
