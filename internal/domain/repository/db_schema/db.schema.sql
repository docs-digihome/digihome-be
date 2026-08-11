CREATE TABLE document_chunks (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  document_name VARCHAR NOT NULL,
  chunk_index INTEGER NOT NULL,
  content TEXT NOT NULL,
  embedding vector(1024),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
