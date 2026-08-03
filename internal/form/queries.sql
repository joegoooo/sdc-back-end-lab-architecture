-- name: Create :one
INSERT INTO forms (title, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetByID :one
SELECT * FROM forms
WHERE id = $1;

-- name: List :many
SELECT * FROM forms
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: Count :one
SELECT count(*) FROM forms;

-- name: ExistsByTitle :one
SELECT EXISTS(SELECT 1 FROM forms WHERE title = $1);

-- name: Update :one
UPDATE forms
SET title       = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: Delete :execrows
DELETE FROM forms
WHERE id = $1;
