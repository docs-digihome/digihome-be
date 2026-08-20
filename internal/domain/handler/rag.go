package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/daffadon/digihome/internal/domain/services"
	"github.com/daffadon/digihome/internal/pkg"
	"github.com/go-chi/chi/v5"
)

type (
	RagHandler interface {
		Seed(w http.ResponseWriter, r *http.Request)
		BatchInsertDocument(w http.ResponseWriter, r *http.Request)
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

// BatchInsertDocument implements [RagHandler].
func (rh *ragHandler) BatchInsertDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		pkg.ReturnError(w, http.StatusBadRequest, err)
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		pkg.ReturnError(w, http.StatusBadRequest, errors.New("no files uploaded"))
		return
	}
	resp, err := rh.rs.BatchInsertDocument(ctx, files)
	if err != nil {
		pkg.ReturnError(w, http.StatusBadRequest, err)
		return
	}
	pkg.ReturnSuccess(w, http.StatusOK, "batch insert success", resp)
}

func RegisterRagRoute(r chi.Router, rh RagHandler) {
	r.Post("/rag/seed", rh.Seed)
	r.Post("/rag/document", rh.BatchInsertDocument)
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
