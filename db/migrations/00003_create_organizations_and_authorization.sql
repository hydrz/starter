-- +goose Up
CREATE TABLE organizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug text NOT NULL UNIQUE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_slug_length CHECK (char_length(slug) BETWEEN 1 AND 100),
    CONSTRAINT organizations_name_length CHECK (char_length(name) BETWEEN 1 AND 120)
);

CREATE TABLE organization_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organization_memberships_org_user_key UNIQUE (organization_id, user_id)
);
CREATE INDEX organization_memberships_user_id_idx ON organization_memberships (user_id);

CREATE TABLE organization_invitations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    email text NOT NULL,
    role text NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    token_digest bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    accepted_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organization_invitations_email_normalized CHECK (email = lower(email)),
    CONSTRAINT organization_invitations_email_length CHECK (char_length(email) BETWEEN 3 AND 320)
);
CREATE INDEX organization_invitations_org_email_idx ON organization_invitations (organization_id, email);

CREATE TABLE casbin_rules (
    id bigserial PRIMARY KEY,
    ptype text NOT NULL DEFAULT '',
    v0 text NOT NULL DEFAULT '',
    v1 text NOT NULL DEFAULT '',
    v2 text NOT NULL DEFAULT '',
    v3 text NOT NULL DEFAULT '',
    v4 text NOT NULL DEFAULT '',
    v5 text NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX casbin_rules_unique_idx ON casbin_rules (ptype, v0, v1, v2, v3, v4, v5);

-- Seed standard role policies into casbin_rules
INSERT INTO casbin_rules (ptype, v0, v1, v2, v3) VALUES
    ('p', 'owner', '*', 'organizations', 'read'),
    ('p', 'owner', '*', 'organizations', 'update'),
    ('p', 'owner', '*', 'organizations', 'delete'),
    ('p', 'owner', '*', 'members', 'read'),
    ('p', 'owner', '*', 'members', 'create'),
    ('p', 'owner', '*', 'members', 'update'),
    ('p', 'owner', '*', 'members', 'delete'),
    ('p', 'owner', '*', 'invitations', 'read'),
    ('p', 'owner', '*', 'invitations', 'create'),
    ('p', 'owner', '*', 'invitations', 'delete'),
    ('p', 'owner', '*', 'announcements', 'read'),
    ('p', 'owner', '*', 'announcements', 'create'),
    ('p', 'owner', '*', 'announcements', 'update'),
    ('p', 'owner', '*', 'announcements', 'delete'),

    ('p', 'admin', '*', 'organizations', 'read'),
    ('p', 'admin', '*', 'organizations', 'update'),
    ('p', 'admin', '*', 'members', 'read'),
    ('p', 'admin', '*', 'members', 'create'),
    ('p', 'admin', '*', 'members', 'update'),
    ('p', 'admin', '*', 'members', 'delete'),
    ('p', 'admin', '*', 'invitations', 'read'),
    ('p', 'admin', '*', 'invitations', 'create'),
    ('p', 'admin', '*', 'invitations', 'delete'),
    ('p', 'admin', '*', 'announcements', 'read'),
    ('p', 'admin', '*', 'announcements', 'create'),
    ('p', 'admin', '*', 'announcements', 'update'),
    ('p', 'admin', '*', 'announcements', 'delete'),

    ('p', 'member', '*', 'organizations', 'read'),
    ('p', 'member', '*', 'members', 'read'),
    ('p', 'member', '*', 'announcements', 'read'),
    ('p', 'member', '*', 'announcements', 'create'),
    ('p', 'member', '*', 'announcements', 'update'),

    ('p', 'viewer', '*', 'organizations', 'read'),
    ('p', 'viewer', '*', 'members', 'read'),
    ('p', 'viewer', '*', 'announcements', 'read')
ON CONFLICT DO NOTHING;

-- Scope announcements by organization
ALTER TABLE announcements ADD COLUMN organization_id uuid REFERENCES organizations (id) ON DELETE CASCADE;

-- Backfill default organization for any pre-existing announcements
DO $$
DECLARE
    default_org_id uuid;
BEGIN
    IF EXISTS (SELECT 1 FROM announcements WHERE organization_id IS NULL) THEN
        INSERT INTO organizations (id, slug, name)
        VALUES (gen_random_uuid(), 'default-org', 'Default Organization')
        RETURNING id INTO default_org_id;

        UPDATE announcements SET organization_id = default_org_id WHERE organization_id IS NULL;
    END IF;
END $$;

ALTER TABLE announcements ALTER COLUMN organization_id SET NOT NULL;
CREATE INDEX announcements_organization_id_idx ON announcements (organization_id, created_at DESC, id DESC);

-- Add foreign key constraint to api_keys.organization_id
ALTER TABLE api_keys ADD CONSTRAINT fk_api_keys_organization FOREIGN KEY (organization_id) REFERENCES organizations (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS fk_api_keys_organization;
DROP INDEX IF EXISTS announcements_organization_id_idx;
ALTER TABLE announcements DROP COLUMN IF EXISTS organization_id;
DROP TABLE IF EXISTS casbin_rules;
DROP TABLE IF EXISTS organization_invitations;
DROP TABLE IF EXISTS organization_memberships;
DROP TABLE IF EXISTS organizations;
