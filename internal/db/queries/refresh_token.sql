-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at,
    created_at,
    ip_address,
    user_agent
) VALUES (
    ?, -- user_id
    ?, -- token_hash
    ?, -- expires_at
    NOW(), -- created_at
    ?, -- ip_address
    ? -- user_agent
)
ON DUPLICATE KEY UPDATE
    token_hash = VALUES(token_hash),
    expires_at = VALUES(expires_at),
    created_at = NOW(),
    ip_address = VALUES(ip_address),
    user_agent = VALUES(user_agent),
    revoked_at = NULL;

-- name: GetRefreshToken :one
SELECT
    user_id,
    token_hash,
    expires_at,
    created_at,
    ip_address,
    user_agent
FROM refresh_tokens
WHERE user_id = ?;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET
    revoked_at = NOW()
WHERE user_id = ?;


