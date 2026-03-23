-- name: CreateProject :execresult
INSERT INTO projects (user_id, name, color_hex) VALUES (?, ?, ?);

-- name: GetProjectsByUserId :many
SELECT projects.id, projects.user_id, projects.name, projects.color_hex, projects.created_at 
FROM projects 
LEFT JOIN shared_projects sp ON projects.id = sp.project_id
WHERE projects.user_id = sqlc.arg(user_id) OR sp.shared_with_user_id = sqlc.arg(user_id)
ORDER BY created_at DESC;

-- name: GetProjectById :one
SELECT id, user_id, name, color_hex, created_at FROM projects WHERE id = ?;

-- name: UpdateProject :exec
UPDATE projects SET name = ?, color_hex = ? WHERE id = ?;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;

-- name: ShareProjectWithUser :exec
INSERT INTO shared_projects (project_id, shared_with_user_id) VALUES (?, ?);

-- name: UnshareProjectWithUser :exec
DELETE FROM shared_projects WHERE project_id = ? AND shared_with_user_id = ?;

-- name: IsProjectSharedWithUser :one
SELECT EXISTS(SELECT 1 FROM shared_projects WHERE project_id = ? AND shared_with_user_id = ?) AS is_shared;
