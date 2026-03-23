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
WHERE tasks.created_by_user_id = sqlc.arg(user_id) OR shared_tasks.shared_with_user_id = sqlc.arg(user_id)
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

-- name: GetTaskById :one
SELECT id, project_id, title, description, task_type, priority, created_by_user_id, created_at
FROM tasks WHERE id = ?;

-- name: IsTaskSharedWithUser :one
SELECT EXISTS(SELECT 1 FROM shared_tasks WHERE task_id = ? AND shared_with_user_id = ?) AS is_shared;

-- name: CompleteTask :exec
INSERT INTO task_logs (task_id, completed_by_user_id) VALUES (?, ?);

-- name: UncompleteTask :exec
DELETE FROM task_logs
WHERE task_id = ? AND completed_by_user_id = ? AND DATE(completed_at) = CURDATE();

-- name: GetTaskCompletions :many
SELECT id, task_id, completed_by_user_id, completed_at
FROM task_logs WHERE task_id = ? ORDER BY completed_at DESC;

-- name: GetCompletionsForWeek :many
SELECT id, task_id, completed_by_user_id, completed_at
FROM task_logs
WHERE completed_by_user_id = sqlc.arg(user_id)
  AND completed_at >= sqlc.arg(start_date)
  AND completed_at < sqlc.arg(end_date)
ORDER BY completed_at DESC;
