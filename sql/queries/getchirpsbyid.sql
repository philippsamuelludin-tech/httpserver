-- name: GetChirpsByID :one
SELECT * FROM chirps
WHERE id = $1;