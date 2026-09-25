-- +goose Up
CREATE TYPE announcement_status AS ENUM ('draft', 'published');

CREATE TABLE announcements (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
    content text NOT NULL CHECK (char_length(content) BETWEEN 1 AND 10000),
    status announcement_status NOT NULL DEFAULT 'draft',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX announcements_created_at_idx ON announcements (created_at DESC, id DESC);
CREATE INDEX announcements_status_created_at_idx ON announcements (status, created_at DESC, id DESC);

-- +goose Down
DROP TABLE announcements;
DROP TYPE announcement_status;
