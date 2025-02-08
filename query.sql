-- name: GetUser :one
SELECT s.*, r.role
FROM shoppers s
         INNER JOIN role r ON s.id = r.user_id
WHERE s.id = $1;

-- name: GetAllUser :many
SELECT s.*, r.role
FROM shoppers s
         INNER JOIN role r ON s.id = r.user_id;

-- name: InsertUser :exec
INSERT INTO shoppers(id, username, passwd, created, lastLogin)
VALUES ($1, $2, $3, $4, $5);

-- name: UpdateUsername :exec
UPDATE shoppers
SET username = $2 AND lastLogin = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdatePassword :exec
UPDATE shoppers
SET passwd = $2 AND lastLogin = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUser :exec
DELETE
FROM shoppers
WHERE id = $1;

-- name: DeleteAllUser :exec
DELETE
FROM shoppers;

-- name: