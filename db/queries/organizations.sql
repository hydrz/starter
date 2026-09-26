-- name: CreateOrganization :one
INSERT INTO organizations (slug, name)
VALUES ($1, $2)
RETURNING id, slug, name, created_at, updated_at;

-- name: GetOrganizationByID :one
SELECT id, slug, name, created_at, updated_at
FROM organizations
WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT id, slug, name, created_at, updated_at
FROM organizations
WHERE slug = $1;

-- name: UpdateOrganization :one
UPDATE organizations
SET name = $2,
    slug = $3,
    updated_at = now()
WHERE id = $1
RETURNING id, slug, name, created_at, updated_at;

-- name: DeleteOrganization :execrows
DELETE FROM organizations
WHERE id = $1;

-- name: ListOrganizationsForUser :many
SELECT o.id, o.slug, o.name, o.created_at, o.updated_at, m.role
FROM organizations o
JOIN organization_memberships m ON o.id = m.organization_id
WHERE m.user_id = $1
ORDER BY o.created_at DESC;

-- name: CreateMembership :one
INSERT INTO organization_memberships (organization_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING id, organization_id, user_id, role, created_at, updated_at;

-- name: GetMembership :one
SELECT id, organization_id, user_id, role, created_at, updated_at
FROM organization_memberships
WHERE organization_id = $1 AND user_id = $2;

-- name: ListMemberships :many
SELECT m.id, m.organization_id, m.user_id, m.role, m.created_at, m.updated_at, u.email
FROM organization_memberships m
JOIN users u ON m.user_id = u.id
WHERE m.organization_id = $1
ORDER BY m.created_at ASC;

-- name: UpdateMembershipRole :one
UPDATE organization_memberships
SET role = $3,
    updated_at = now()
WHERE organization_id = $1 AND user_id = $2
RETURNING id, organization_id, user_id, role, created_at, updated_at;

-- name: DeleteMembership :execrows
DELETE FROM organization_memberships
WHERE organization_id = $1 AND user_id = $2;

-- name: CountOrganizationOwners :one
SELECT count(*)
FROM organization_memberships
WHERE organization_id = $1 AND role = 'owner';

-- name: CreateInvitation :one
INSERT INTO organization_invitations (organization_id, email, role, token_digest, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, organization_id, email, role, token_digest, expires_at, accepted_at, revoked_at, created_at;

-- name: GetInvitationByID :one
SELECT id, organization_id, email, role, token_digest, expires_at, accepted_at, revoked_at, created_at
FROM organization_invitations
WHERE id = $1;

-- name: GetInvitationByDigest :one
SELECT id, organization_id, email, role, token_digest, expires_at, accepted_at, revoked_at, created_at
FROM organization_invitations
WHERE token_digest = $1;

-- name: ListInvitations :many
SELECT id, organization_id, email, role, token_digest, expires_at, accepted_at, revoked_at, created_at
FROM organization_invitations
WHERE organization_id = $1
  AND accepted_at IS NULL
  AND revoked_at IS NULL
  AND expires_at > now()
ORDER BY created_at DESC;

-- name: AcceptInvitation :one
UPDATE organization_invitations
SET accepted_at = now()
WHERE id = $1
  AND accepted_at IS NULL
  AND revoked_at IS NULL
  AND expires_at > now()
RETURNING id, organization_id, email, role, token_digest, expires_at, accepted_at, revoked_at, created_at;

-- name: RevokeInvitation :execrows
UPDATE organization_invitations
SET revoked_at = now()
WHERE id = $1
  AND organization_id = $2
  AND accepted_at IS NULL
  AND revoked_at IS NULL;
