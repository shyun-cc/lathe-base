-- +goose Up
CREATE TABLE outsource_returns (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    return_no text NOT NULL UNIQUE CHECK (btrim(return_no) <> ''),
    job_id bigint NOT NULL REFERENCES outsource_jobs (id),
    received_on date NOT NULL,
    accepted_milli bigint NOT NULL CHECK (accepted_milli >= 0),
    rejected_milli bigint NOT NULL CHECK (rejected_milli >= 0),
    CHECK (accepted_milli > 0 OR rejected_milli > 0)
);

CREATE INDEX outsource_returns_job_idx ON outsource_returns (job_id);

-- +goose Down
DROP TABLE outsource_returns;
