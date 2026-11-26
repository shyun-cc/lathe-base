-- +goose Up
CREATE TABLE printing_templates (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code text NOT NULL CHECK (btrim(code) <> ''),
    version integer NOT NULL CHECK (version > 0),
    name text NOT NULL CHECK (btrim(name) <> ''),
    document_type text NOT NULL CHECK (btrim(document_type) <> ''),
    configuration jsonb NOT NULL CHECK (jsonb_typeof(configuration) = 'object'),
    active boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (document_type, code, version)
);

CREATE UNIQUE INDEX printing_templates_active_idx ON printing_templates (document_type, code) WHERE active;

-- +goose Down
DROP TABLE printing_templates;
