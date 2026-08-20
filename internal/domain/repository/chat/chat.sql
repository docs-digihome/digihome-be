-- name: CreateNewMessage :exec
INSERT INTO messages (role,content,embedding)
VALUES ($1,$2,$3);

-- name: GetLatestMessages :many
SELECT
  role,
  content,
  created_at
FROM messages
ORDER BY created_at DESC
LIMIT $1;

-- name: GetMessagesBefore :many
SELECT 
  role, 
  content,
  created_at
FROM messages
WHERE created_at < $1
ORDER BY id DESC
LIMIT $2;

-- name: SearchSimilarMessages :many
SELECT
  role,
  content,
  created_at
FROM messages
WHERE embedding IS NOT NULL
ORDER BY embedding <=> $1
LIMIT $2;
