package constant

const (
	DEFAULT_EMBED_MODEL    = "bge-m3"
	DEFAULT_EMBED_ENDOPINT = "http://localhost:11434/api/embed"
)

// Deprecated: use viper keys llm.embed.model / llm.embed.endpoint instead.
// Kept for fallback when config is absent.
