-- name: insertClient :exec
INSERT INTO clients (
    client_id,
    client_secret_hash,
    client_type,
    app_name,
) VALUES (
    ?, -- client_id
    ?, -- client_secret_hash
    ?, -- client_type
    ?, -- app_name
)


-- name: removeClient :exec
DELETE FROM clients
WHERE client_id = ?

