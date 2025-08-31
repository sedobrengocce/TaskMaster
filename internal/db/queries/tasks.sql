-- name: CreateTask :exec
INSERT INTO tasks (project_id, title, description, task_type, priority, created_by_user_id) 
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetTaskListByProjectId :many
SELECT id, project_id, title, description, task_type, priority, created_by_user_id, created_at 
FROM tasks 
WHERE project_id = ? 
ORDER BY created_at DESC;

-- name: GetTasksByUserId :many
SELECT tasks.id, tasks.project_id, tasks.title, tasks.description, tasks.task_type, tasks.priority, tasks.created_by_user_id, tasks.created_at
FROM tasks
LEFT JOIN shared_tasks ON tasks.id = shared_tasks.task_id
WHERE tasks.created_by_user_id = ? OR shared_tasks.shared_with_user_id = ?
ORDER BY tasks.created_at DESC;

-- name: UpdateTask :exec
UPDATE tasks
SET project_id = ?, title = ?, description = ?, task_type = ?, priority = ?
WHERE id = ?;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

-- name: ShareTaskWithUser :exec
INSERT INTO shared_tasks (task_id, shared_with_user_id) VALUES (?, ?);

-- name: UnshareTaskWithUser :exec
DELETE FROM shared_tasks WHERE task_id = ? AND shared_with_user_id = ?;


