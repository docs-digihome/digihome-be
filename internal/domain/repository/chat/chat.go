package chat_repository

import "github.com/jackc/pgx/v5/pgxpool"

func NewChatQueries(pool *pgxpool.Pool) *Queries {
	return New(pool)
}
