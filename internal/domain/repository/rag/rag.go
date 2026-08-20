package rag_repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRAGQueries(pool *pgxpool.Pool) *Queries {
	return New(pool)
}
