-- +goose Up
CREATE TABLE production_reports (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    report_no text NOT NULL UNIQUE CHECK (btrim(report_no) <> ''),
    work_order_id bigint NOT NULL REFERENCES production_work_orders (id),
    operation_id bigint NOT NULL REFERENCES master_operations (id),
    employee_id bigint NOT NULL REFERENCES master_employees (id),
    operation_name text NOT NULL CHECK (btrim(operation_name) <> ''),
    reported_on date NOT NULL,
    quantity_milli bigint NOT NULL CHECK (quantity_milli > 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX production_reports_capacity_idx ON production_reports (work_order_id, operation_id);

-- +goose Down
DROP TABLE production_reports;
