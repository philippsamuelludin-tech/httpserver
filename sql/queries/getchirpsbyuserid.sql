-- name: GetChirpsByUserID :one
SELECT * FROM chirps
WHERE user_id = $1
ORDER BY created_at;