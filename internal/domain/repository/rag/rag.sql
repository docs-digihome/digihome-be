-- name: SearchDocumentChunks :many
SELECT
    id,
    page_number,
    chunk_index,
    content,
    created_at,
    1 - (embedding <=> $1) AS similarity
FROM document_chunks
ORDER BY embedding <=> $1
LIMIT $2;
