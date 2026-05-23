-- name: LookUpUserByEmail :one
SELECT * FROM users
WHERE email = $1;