-- name: GetUser :one
SELECT *
FROM shoppers
WHERE id = $1;