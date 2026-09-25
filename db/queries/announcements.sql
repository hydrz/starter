-- name: CountAnnouncements :one
SELECT count(*)
FROM announcements
WHERE sqlc.narg('status')::announcement_status IS NULL
   OR status = sqlc.narg('status')::announcement_status;

-- name: ListAnnouncements :many
SELECT id, title, content, status, created_at, updated_at
FROM announcements
WHERE sqlc.narg('status')::announcement_status IS NULL
   OR status = sqlc.narg('status')::announcement_status
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetAnnouncement :one
SELECT id, title, content, status, created_at, updated_at
FROM announcements
WHERE id = $1;

-- name: CreateAnnouncement :one
INSERT INTO announcements (title, content, status)
VALUES ($1, $2, $3)
RETURNING id, title, content, status, created_at, updated_at;

-- name: UpdateAnnouncement :one
UPDATE announcements
SET title = $2,
    content = $3,
    status = $4,
    updated_at = now()
WHERE id = $1
RETURNING id, title, content, status, created_at, updated_at;

-- name: DeleteAnnouncement :execrows
DELETE FROM announcements
WHERE id = $1;
