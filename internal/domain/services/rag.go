package services

import (
	"log/slog"

	rag_repository "github.com/daffadon/digihome/internal/domain/repository/rag"
)

type (
	RagService interface {
		Seed() error
	}
	ragService struct {
		slog *slog.Logger
		rr   *rag_repository.Queries
	}
)

func NewRagService(slog *slog.Logger, rr *rag_repository.Queries) RagService {
	return &ragService{
		slog: slog,
		rr:   rr,
	}
}

// Seed implements [RagService].
func (r *ragService) Seed() error {
	panic("unimplemented")
}
