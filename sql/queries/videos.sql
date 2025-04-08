-- name: NewVideo :one
INSERT INTO video(name, uploaded)
     VALUES (?, ?)
  RETURNING *;

-- name: Ping :one
SELECT 1;
