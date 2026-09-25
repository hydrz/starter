-- +goose Up
CREATE TABLE system_settings (
    key text PRIMARY KEY CHECK (key <> ''),
    value jsonb NOT NULL,
    description text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE system_settings;
