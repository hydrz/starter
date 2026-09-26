-- name: ListCasbinRules :many
SELECT id, ptype, v0, v1, v2, v3, v4, v5
FROM casbin_rules
ORDER BY id ASC;

-- name: InsertCasbinRule :one
INSERT INTO casbin_rules (ptype, v0, v1, v2, v3, v4, v5)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (ptype, v0, v1, v2, v3, v4, v5) DO NOTHING
RETURNING id, ptype, v0, v1, v2, v3, v4, v5;

-- name: DeleteCasbinRule :execrows
DELETE FROM casbin_rules
WHERE ptype = $1
  AND v0 = $2
  AND v1 = $3
  AND v2 = $4
  AND v3 = $5
  AND v4 = $6
  AND v5 = $7;

-- name: DeleteCasbinRulesByPtype :execrows
DELETE FROM casbin_rules
WHERE ptype = $1;

-- name: DeleteCasbinRulesByDomain :execrows
DELETE FROM casbin_rules
WHERE (ptype = 'p' AND v1 = $1)
   OR (ptype = 'g' AND v2 = $1);
