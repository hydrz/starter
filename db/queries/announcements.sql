-- name: CountAnnouncements :one
SELECT count(*)
FROM announcements
WHERE organization_id = $1
  AND (sqlc.narg('status')::announcement_status IS NULL
       OR status = sqlc.narg('status')::announcement_status);

-- name: ListAnnouncements :many
SELECT id, organization_id, title, content, status, created_at, updated_at
FROM announcements
WHERE organization_id = $1
  AND (sqlc.narg('status')::announcement_status IS NULL
       OR status = sqlc.narg('status')::announcement_status)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetAnnouncement :one
SELECT id, organization_id, title, content, status, created_at, updated_at
FROM announcements
WHERE id = $1 AND organization_id = $2;

-- name: CreateAnnouncement :one
INSERT INTO announcements (organization_id, title, content, status)
VALUES ($1, $2, $3, $4)
RETURNING id, organization_id, title, content, status, created_at, updated_at;

-- name: UpdateAnnouncement :one
UPDATE announcements
SET title = $3,
    content = $4,
    status = $5,
    updated_at = now()
WHERE id = $1 AND organization_id = $2
RETURNING id, organization_id, title, content, status, created_at, updated_at;

-- name: DeleteAnnouncement :execrows
DELETE FROM announcements
WHERE id = $1 AND organization_id = $2;
