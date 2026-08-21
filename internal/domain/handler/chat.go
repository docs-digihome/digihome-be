package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/daffadon/digihome/internal/domain/services"
	"github.com/daffadon/digihome/internal/pkg"
	"github.com/daffadon/digihome/internal/schema"
	"github.com/go-chi/chi/v5"
)

type (
	ChatHandler interface {
		GetLatestChatWithPagination(w http.ResponseWriter, r *http.Request)
		Chat(w http.ResponseWriter, r *http.Request)
	}
	chatHandler struct {
		slog *slog.Logger
		cs   services.ChatService
	}
)

func RegisterChatRoute(r chi.Router, ch ChatHandler) {
	r.Get("/chat", ch.GetLatestChatWithPagination)
	r.Post("/chat", ch.Chat)
}

func NewChatHandler(slog *slog.Logger, cs services.ChatService) ChatHandler {
	return &chatHandler{
		slog: slog,
		cs:   cs,
	}
}

// Chat implements [ChatHandler].
func (c *chatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req, sc, err, ok := pkg.DecodeAndValidateBody[schema.ChatRequest](w, r, c.slog)
	if !ok {
		pkg.ReturnError(w, sc, err)
		return
	}
	resp, err := c.cs.Chat(ctx, req.TextPrompt)
	if err != nil {
		pkg.ReturnError(w, http.StatusInternalServerError, err)
		return
	}
	pkg.ReturnSuccess(w, http.StatusOK, "send prompt success", resp)
}

// GetLatestChatWithPagination implements [ChatHandler].
func (c *chatHandler) GetLatestChatWithPagination(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var dateTime time.Time
	beforeTime := r.URL.Query().Get("before")
	if beforeTime != "" {
		parsed, err := time.Parse(time.RFC3339, beforeTime)
		if err == nil {
			dateTime = parsed
		}
	}
	resp, err := c.cs.GetLatestChat(ctx, dateTime)
	if err != nil {
		c.slog.Error("error endpoint", "err", err)
		pkg.ReturnError(w, http.StatusInternalServerError, err)
		return
	}
	pkg.ReturnSuccess(w, http.StatusOK, "get chat success", resp)
}
