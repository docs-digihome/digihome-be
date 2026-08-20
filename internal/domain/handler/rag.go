package handler

import (
	"log/slog"
	"net/http"

	"github.com/daffadon/digihome/internal/domain/services"
	"github.com/daffadon/digihome/internal/pkg"
	"github.com/go-chi/chi/v5"
)

type (
	RagHandler interface {
		Seed(w http.ResponseWriter, r *http.Request)
	}
	ragHandler struct {
		slog *slog.Logger
		rs   services.RagService
	}
)

func NewRagHandler(slog *slog.Logger, rs services.RagService) RagHandler {
	return &ragHandler{
		slog: slog,
		rs:   rs,
	}
}

func RegisterRagRoute(r chi.Router, rh RagHandler) {
	r.Post("/rag/seed", rh.Seed)
}

// Seed implements [RagHandler].
func (rh *ragHandler) Seed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := rh.rs.Seed(ctx); err != nil {
		pkg.ReturnError(w, http.StatusInternalServerError, err)
		return
	}
	pkg.ReturnSuccess(w, http.StatusCreated, "seed success", nil)
}
