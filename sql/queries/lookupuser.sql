-- name: LookUpUser :one
SELECT * FROM users
WHERE email = $1;