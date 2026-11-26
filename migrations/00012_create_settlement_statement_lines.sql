-- +goose Up
CREATE TABLE settlement_statement_lines (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    statement_id bigint NOT NULL,
    party_id bigint NOT NULL,
    direction text NOT NULL,
    currency text NOT NULL,
    shipment_id bigint UNIQUE,
    outsource_job_id bigint UNIQUE,
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    FOREIGN KEY (statement_id, party_id, direction, currency)
        REFERENCES settlement_statements (id, party_id, direction, currency),
    FOREIGN KEY (shipment_id, party_id, currency) REFERENCES shipping_shipments (id, customer_id, currency),
    FOREIGN KEY (outsource_job_id, party_id, currency) REFERENCES outsource_jobs (id, supplier_id, currency),
    CHECK (
        (direction = 'receivable' AND shipment_id IS NOT NULL AND outsource_job_id IS NULL)
        OR (direction = 'payable' AND shipment_id IS NULL AND outsource_job_id IS NOT NULL)
    )
);

CREATE INDEX settlement_statement_lines_statement_idx ON settlement_statement_lines (statement_id);

-- +goose Down
DROP TABLE settlement_statement_lines;
