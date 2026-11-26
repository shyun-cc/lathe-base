-- +goose Up
CREATE TABLE outsource_jobs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    job_no text NOT NULL UNIQUE CHECK (btrim(job_no) <> ''),
    work_order_id bigint NOT NULL REFERENCES production_work_orders (id),
    supplier_id bigint NOT NULL REFERENCES master_parties (id),
    operation_name text NOT NULL CHECK (btrim(operation_name) <> ''),
    quantity_milli bigint NOT NULL CHECK (quantity_milli > 0),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    status text NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'sent', 'closed', 'cancelled')),
    sent_on date,
    due_on date,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, supplier_id, currency)
);

CREATE INDEX outsource_jobs_work_order_idx ON outsource_jobs (work_order_id);
CREATE INDEX outsource_jobs_supplier_idx ON outsource_jobs (supplier_id);

-- +goose Down
DROP TABLE outsource_jobs;
