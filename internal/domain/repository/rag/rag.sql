-- name: GetRegisteredDocuments :many
SELECT
  DISTINCT document_name
FROM document_chunks;

-- name: SearchDocumentChunks :many
SELECT
    id,
    document_name,
    chunk_index,
    content,
    created_at,
    1 - (embedding <=> $1) AS similarity
FROM document_chunks
ORDER BY embedding <=> $1
LIMIT $2;

-- name: InsertDocumentChunks :exec
INSERT INTO document_chunks (document_name, chunk_index, content, embedding)
VALUES ($1,$2,$3,$4);
