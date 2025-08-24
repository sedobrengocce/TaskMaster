-- name: CreateProject :exec
INSERT INTO projects (user_id, name, color_hex) VALUES (?, ?, ?);

-- name: GetProjectsByUserId :many
SELECT id, user_id, name, color_hex, created_at FROM projects WHERE user_id = ? ORDER BY created_at DESC;

-- name: GetProjectById :one
SELECT id, user_id, name, color_hex, created_at FROM projects WHERE id = ?;

-- name: UpdateProject :exec
UPDATE projects SET name = ?, color_hex = ? WHERE id = ?;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;
