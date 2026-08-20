package services

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/daffadon/digihome/internal/constant"
	chat_repository "github.com/daffadon/digihome/internal/domain/repository/chat"
	rag_repository "github.com/daffadon/digihome/internal/domain/repository/rag"
	"github.com/daffadon/digihome/internal/pkg"
	"github.com/daffadon/digihome/internal/schema"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
	"github.com/spf13/viper"
)

const (
	roleUser      = "user"
	roleAssistant = "assistant"

	systemPrompt = "You are DigiHome, a helpful and concise smart-home assistant. Answer based on the conversation history and the provided document excerpts when relevant. Be concise: reply in a few short sentences, without preamble or filler."
)

type (
	ChatService interface {
		Chat(ctx context.Context, text string) (schema.ChatResponse, error)
		GetLatestChart(ctx context.Context, beforeTime time.Time) ([]schema.MessageResponse, error)
	}
	chatService struct {
		slog *slog.Logger
		rr   *chat_repository.Queries
		rgrr *rag_repository.Queries
	}

	chatConfig struct {
		endpoint           string
		model              string
		numCtx             int
		reserveReplyTokens int
		topKHistory        int
		recentMessages     int
		topKDocuments      int
		maxTokens          int
	}
)

func NewChatService(slog *slog.Logger, rr *chat_repository.Queries, rgrr *rag_repository.Queries) ChatService {
	return &chatService{
		slog: slog,
		rr:   rr,
		rgrr: rgrr,
	}
}

// Chat implements [ChatService].
func (c *chatService) Chat(ctx context.Context, text string) (schema.ChatResponse, error) {
	cfg := chatConfigFromViper()

	promptEmbedding, err := pkg.Embed(ctx, constant.DEFAULT_EMBED_MODEL, text)
	if err != nil {
		return schema.ChatResponse{}, err
	}
	promptVector := pgvector.NewVector(promptEmbedding)

	docChunks, err := c.rgrr.SearchDocumentChunks(ctx, rag_repository.SearchDocumentChunksParams{
		Embedding: promptVector,
		Limit:     int32(cfg.topKDocuments),
	})
	if err != nil {
		return schema.ChatResponse{}, err
	}

	similar, err := c.rr.SearchSimilarMessages(ctx, chat_repository.SearchSimilarMessagesParams{
		Embedding: promptVector,
		Limit:     int32(cfg.topKHistory),
	})
	if err != nil {
		return schema.ChatResponse{}, err
	}

	recent, err := c.rr.GetLatestMessages(ctx, int32(cfg.recentMessages))
	if err != nil {
		return schema.ChatResponse{}, err
	}

	if err := c.rr.CreateNewMessage(ctx, chat_repository.CreateNewMessageParams{
		Role:      roleUser,
		Content:   text,
		Embedding: promptVector,
	}); err != nil {
		return schema.ChatResponse{}, err
	}

	history := mergeHistory(recent, similar)
	messages := buildMessages(cfg, history, docChunks, text)

	reply, err := pkg.Chat(ctx, cfg.endpoint, cfg.model, messages, cfg.numCtx, cfg.maxTokens)
	if err != nil {
		return schema.ChatResponse{}, err
	}

	if strings.TrimSpace(reply) == "" {
		c.slog.Warn("empty assistant reply, not persisted")
		return schema.ChatResponse{Reply: reply, Documents: documentSources(docChunks)}, nil
	}

	replyEmbedding, err := pkg.Embed(ctx, constant.DEFAULT_EMBED_MODEL, reply)
	if err != nil {
		c.slog.Warn("embedding assistant reply failed, reply not persisted",
			"error", err,
		)
		return schema.ChatResponse{Reply: reply, Documents: documentSources(docChunks)}, nil
	}
	if err := c.rr.CreateNewMessage(ctx, chat_repository.CreateNewMessageParams{
		Role:      roleAssistant,
		Content:   reply,
		Embedding: pgvector.NewVector(replyEmbedding),
	}); err != nil {
		return schema.ChatResponse{}, err
	}

	return schema.ChatResponse{Reply: reply, Documents: documentSources(docChunks)}, nil
}

// GetLatestChart implements [ChatService].
func (c *chatService) GetLatestChart(ctx context.Context, beforeTime time.Time) ([]schema.MessageResponse, error) {
	const limit = 20

	if beforeTime.IsZero() {
		rows, err := c.rr.GetLatestMessages(ctx, limit)
		if err != nil {
			return nil, err
		}
		return mapLatestMessages(rows), nil
	}

	rows, err := c.rr.GetMessagesBefore(ctx, chat_repository.GetMessagesBeforeParams{
		CreatedAt: pgtype.Timestamptz{Time: beforeTime, Valid: true},
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}
	return mapMessagesBefore(rows), nil
}

// chatConfigFromViper reads the llm.chat settings from the app config.
func chatConfigFromViper() chatConfig {
	cfg := chatConfig{
		endpoint:           viper.GetString("llm.chat.endpoint"),
		model:              viper.GetString("llm.chat.model"),
		numCtx:             viper.GetInt("llm.chat.num_ctx"),
		reserveReplyTokens: viper.GetInt("llm.chat.reserve_reply_tokens"),
		topKHistory:        viper.GetInt("llm.chat.top_k_history"),
		recentMessages:     viper.GetInt("llm.chat.recent_messages"),
		topKDocuments:      viper.GetInt("llm.chat.top_k_documents"),
		maxTokens:          viper.GetInt("llm.chat.max_tokens"),
	}

	if cfg.endpoint == "" {
		cfg.endpoint = "http://localhost:11434/api/chat"
	}
	if cfg.model == "" {
		cfg.model = "qwen3.5:0.8b"
	}
	if cfg.numCtx <= 0 {
		cfg.numCtx = 16384
	}
	if cfg.reserveReplyTokens <= 0 {
		cfg.reserveReplyTokens = 4096
	}
	if cfg.topKHistory <= 0 {
		cfg.topKHistory = 10
	}
	if cfg.recentMessages <= 0 {
		cfg.recentMessages = 5
	}
	if cfg.topKDocuments <= 0 {
		cfg.topKDocuments = 3
	}
	if cfg.maxTokens <= 0 {
		cfg.maxTokens = 256
	}
	return cfg
}

// mergeHistory combines the recent messages with the semantically similar ones,
// dedupes by creation time and returns them in chronological order.
func mergeHistory(recent []chat_repository.GetLatestMessagesRow, similar []chat_repository.SearchSimilarMessagesRow) []schema.MessageResponse {
	seen := make(map[int64]struct{}, len(recent)+len(similar))
	history := make([]schema.MessageResponse, 0, len(recent)+len(similar))

	add := func(role, content string, createdAt time.Time) {
		key := createdAt.UnixNano()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		history = append(history, schema.MessageResponse{
			Role:      role,
			Content:   content,
			CreatedAt: createdAt,
		})
	}

	for _, row := range recent {
		add(row.Role, row.Content, row.CreatedAt.Time)
	}
	for _, row := range similar {
		add(row.Role, row.Content, row.CreatedAt.Time)
	}

	slices.SortFunc(history, func(a, b schema.MessageResponse) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return history
}

// buildMessages assembles the prompt sent to the LLM. Retrieved document excerpts
// get priority on the context budget, then the most recent history fills the rest.
func buildMessages(cfg chatConfig, history []schema.MessageResponse, documents []rag_repository.SearchDocumentChunksRow, prompt string) []pkg.ChatMessage {
	available := cfg.numCtx - cfg.reserveReplyTokens - estimateTokens(systemPrompt) - estimateTokens(prompt)

	messages := make([]pkg.ChatMessage, 0, len(history)+3)
	messages = append(messages, pkg.ChatMessage{Role: "system", Content: systemPrompt})

	if context := buildDocumentContext(documents, &available); context != "" {
		messages = append(messages, pkg.ChatMessage{Role: "system", Content: context})
	}

	selected := make([]schema.MessageResponse, 0, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		cost := estimateTokens(history[i].Content)
		if available-cost < 0 {
			break
		}
		available -= cost
		selected = append(selected, history[i])
	}
	slices.Reverse(selected)

	for _, msg := range selected {
		messages = append(messages, pkg.ChatMessage{Role: msg.Role, Content: msg.Content})
	}
	messages = append(messages, pkg.ChatMessage{Role: roleUser, Content: prompt})
	return messages
}

// buildDocumentContext formats the retrieved chunks into a labeled context block,
// deducting their cost from the remaining budget.
func buildDocumentContext(documents []rag_repository.SearchDocumentChunksRow, available *int) string {
	var b strings.Builder
	for i, chunk := range documents {
		label := fmt.Sprintf("[%d] (%s, chunk %d)\n%s\n", i+1, chunk.DocumentName, chunk.ChunkIndex, chunk.Content)
		cost := estimateTokens(label)
		if *available-cost < 0 {
			break
		}
		*available -= cost
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(label)
	}
	if b.Len() == 0 {
		return ""
	}
	return "Relevant document excerpts:\n" + b.String()
}

// documentSources returns the unique document names used as context.
func documentSources(documents []rag_repository.SearchDocumentChunksRow) []string {
	seen := make(map[string]struct{}, len(documents))
	sources := make([]string, 0, len(documents))
	for _, chunk := range documents {
		if _, ok := seen[chunk.DocumentName]; ok {
			continue
		}
		seen[chunk.DocumentName] = struct{}{}
		sources = append(sources, chunk.DocumentName)
	}
	return sources
}

// estimateTokens approximates the token count of a text using a 4 chars/token heuristic.
func estimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// mapLatestMessages converts GetLatestMessagesRow into the shared response type.
func mapLatestMessages(rows []chat_repository.GetLatestMessagesRow) []schema.MessageResponse {
	resp := make([]schema.MessageResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, schema.MessageResponse{
			Role:      row.Role,
			Content:   row.Content,
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return resp
}

// mapMessagesBefore converts GetMessagesBeforeRow into the shared response type.
func mapMessagesBefore(rows []chat_repository.GetMessagesBeforeRow) []schema.MessageResponse {
	resp := make([]schema.MessageResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, schema.MessageResponse{
			Role:      row.Role,
			Content:   row.Content,
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return resp
}
