-- name: InsertClient :exec
INSERT INTO clients (
    client_id,
    client_secret_hash,
    client_type,
    app_name
) VALUES (
    ?, -- client_id
    ?, -- client_secret_hash
    ?, -- client_type
    ? -- app_name
);

-- name: GetClientByClientID :one
SELECT
    client_id,
    client_secret_hash,
    client_type,
    app_name
FROM clients
WHERE client_id = ?;

