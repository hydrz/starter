-- name: GetSystemSetting :one
SELECT key, value, description, updated_at
FROM system_settings
WHERE key = $1;

-- name: ListSystemSettings :many
SELECT key, value, description, updated_at
FROM system_settings
ORDER BY key;

-- name: UpsertSystemSetting :one
INSERT INTO system_settings (key, value, description)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_at = now()
RETURNING key, value, description, updated_at;

-- name: DeleteSystemSetting :execrows
DELETE FROM system_settings
WHERE key = $1;
