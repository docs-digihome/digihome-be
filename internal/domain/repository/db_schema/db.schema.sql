CREATE TABLE document_chunks (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  page_number INTEGER,
  chunk_index INTEGER NOT NULL,
  content TEXT NOT NULL,
  embedding vector(1024),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
