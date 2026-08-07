package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

func NewPostgresqlConn(slog *slog.Logger) *pgxpool.Pool {
	protocol := viper.GetString("database.sql.protocol")
	host := viper.GetString("database.sql.host")
	user := viper.GetString("database.sql.user")
	password := viper.GetString("database.sql.password")
	port := viper.GetString("database.sql.port")
	dbname := viper.GetString("database.sql.name")
	sslmode := viper.GetString("database.sql.sslmode")

	dsn := fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=%s", protocol, user, password, host, port, dbname, sslmode)
	if dsn == "" {
		slog.Error("Database configuration is not set")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		slog.Error("Database configuration is not set", "error", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("Failed to ping PostgreSQL", "error", err)
	}
	return pool
}
