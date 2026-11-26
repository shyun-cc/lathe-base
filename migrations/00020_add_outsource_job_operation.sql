-- +goose Up
ALTER TABLE outsource_jobs ADD COLUMN operation_id bigint REFERENCES master_operations (id);

-- +goose Down
ALTER TABLE outsource_jobs DROP COLUMN operation_id;
