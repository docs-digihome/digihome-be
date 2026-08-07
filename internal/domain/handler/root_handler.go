package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type (
	RootHandler interface {
		GetHealth(w http.ResponseWriter, r *http.Request)
	}
	rootHandler struct {
		slog *slog.Logger
	}
)

func RegisterRootRoute(r chi.Router, rh RootHandler) {
	r.Get("/health", rh.GetHealth)
}

func NewRootHandler(slog *slog.Logger) RootHandler {
	return &rootHandler{
		slog: slog,
	}
}

// GetHealth implements [RootHandler].
func (rh *rootHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	rh.slog.Info("Health is okay")
	// fmt.Println("halo")
}
