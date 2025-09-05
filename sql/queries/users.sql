-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1, $2
)
RETURNING *;

-- name: DeleteUsers :exec
DELETE FROM users;


-- name: GetUser :one
SELECT * from users
WHERE email = $1;

-- name: UpdateUserByID :one
UPDATE users
SET email = $1, hashed_password = $2
WHERE id = $3
RETURNING id, created_at, email, is_chirpy_red;

-- name: UpgradeUserToRed :execrows
UPDATE users
SET is_chirpy_red = true
WHERE id = $1;