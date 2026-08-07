package logger

import (
	"log/slog"
	"os"
)

func NewSlogLogger() *slog.Logger {
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, nil),
	)
	return logger
}
