-- name: CreateNewMessage :exec
INSERT INTO messages (role,content,embedding)
VALUES ($1,$2,$3);

-- name: CreateNewMessageReturningID :one
INSERT INTO messages (role,content,embedding)
VALUES ($1,$2,$3)
RETURNING id;

-- name: GetLatestMessages :many
SELECT *
FROM (
  SELECT
    id,
    role,
    content,
    created_at
  FROM messages
  ORDER BY created_at DESC
  LIMIT $1
) AS latest
ORDER BY created_at ASC;

-- name: GetMessagesBefore :many
SELECT *
FROM (
  SELECT
    id,
    role,
    content,
    created_at
  FROM messages
  WHERE created_at < $1
  ORDER BY created_at DESC
  LIMIT $2
) AS messages_page
ORDER BY created_at ASC;

-- name: SearchSimilarMessages :many
SELECT
  role,
  content,
  created_at
FROM messages
WHERE embedding IS NOT NULL
ORDER BY embedding <=> $1
LIMIT $2;

-- name: InsertMessageDocumentChunk :exec
INSERT INTO message_document_chunks (message_id, chunk_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetDocumentNamesByMessageID :many
SELECT dc.document_name, dc.link
FROM message_document_chunks mdc
JOIN document_chunks dc ON dc.id = mdc.chunk_id
WHERE mdc.message_id = $1
ORDER BY dc.document_name ASC;
