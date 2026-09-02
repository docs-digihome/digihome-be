package constant

const (
	DEFAULT_EMBED_MODEL    = "bge-m3"
	DEFAULT_EMBED_ENDOPINT = "http://localhost:11434/api/embed"
	// DEFAULT_EMBED_ENDPOINT is the correctly spelled alias for DEFAULT_EMBED_ENDOPINT.
	DEFAULT_EMBED_ENDPOINT = "http://localhost:11434/api/embed"
)

// Deprecated: use viper keys llm.embed.model / llm.embed.endpoint instead.
// Kept for fallback when config is absent.
