package main

import (
	"log/slog"

	"github.com/daffadon/digihome/cmd/server"
	"github.com/daffadon/digihome/config/db"
	"github.com/daffadon/digihome/config/env"
	"github.com/daffadon/digihome/config/logger"
	"github.com/daffadon/digihome/config/router"
	s3_config "github.com/daffadon/digihome/config/s3"
	"github.com/daffadon/digihome/internal/domain/handler"
	chat_repository "github.com/daffadon/digihome/internal/domain/repository/chat"
	rag_repository "github.com/daffadon/digihome/internal/domain/repository/rag"
	s3_repository "github.com/daffadon/digihome/internal/domain/repository/s3"
	"github.com/daffadon/digihome/internal/domain/services"
	s3_infra "github.com/daffadon/digihome/internal/infra/s3"
	"github.com/daffadon/digihome/internal/pkg"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func main() {
	env.Load()
	fx.New(
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
		fx.Provide(
			logger.NewSlogLogger,
			pkg.NewMarkdownNormalizer,
			pkg.NewMarkdownChunker,
			s3_config.NewRustfsConnection,
			s3_infra.NewRustfsInfra,
			db.NewPostgresqlConn,
			rag_repository.NewRAGQueries,
			chat_repository.NewChatQueries,
			s3_repository.NewS3Repository,
			services.NewRagService,
			services.NewChatService,
			handler.NewRootHandler,
			handler.NewRagHandler,
			handler.NewChatHandler,
			router.NewHTTPChi,
		),
		fx.Invoke(
			handler.RegisterRootRoute,
			handler.RegisterRagRoute,
			handler.RegisterChatRoute,
			server.RunHTTPServer,
		),
	).Run()
}
