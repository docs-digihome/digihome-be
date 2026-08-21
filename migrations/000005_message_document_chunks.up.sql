CREATE TABLE message_document_chunks (
  message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  chunk_id UUID NOT NULL REFERENCES document_chunks(id) ON DELETE CASCADE,
  PRIMARY KEY (message_id, chunk_id)
);
CREATE INDEX idx_message_document_chunks_message_id ON message_document_chunks(message_id);
