-- name: CreateUser :exec
-- Crea un nuovo utente e restituisce l'utente appena creato.
INSERT INTO users (
    email,
    password_hash
) VALUES (
    ?, ?
);

-- name: GetUserByID :one
-- Recupera un utente dal suo ID.
SELECT * FROM users
WHERE id = ? LIMIT 1;

-- name: UpdateUser :exec
-- Aggiorna un utente esistente e restituisce l'utente aggiornato.
UPDATE users
SET
    email = ?,
    password_hash = ?
WHERE id = ?;

-- name: DeleteUser :exec
-- Elimina un utente e restituisce l'utente eliminato.
DELETE FROM users
WHERE id = ?;

-- name: GetUserByEmailAndPassword :one
-- Recupera un utente dal suo indirizzo email e dalla password.
SELECT * FROM users
WHERE email = ? AND password_hash = ?
LIMIT 1;

-- name: GetUserByTerm :many
-- Recupera gli utenti che corrispondono a un termine di ricerca.
SELECT * FROM users
WHERE email ILIKE '%' || ? || '%'
