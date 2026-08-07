package main

import (
	"log/slog"

	"github.com/daffadon/digihome/cmd/server"
	"github.com/daffadon/digihome/config/db"
	"github.com/daffadon/digihome/config/env"
	"github.com/daffadon/digihome/config/logger"
	"github.com/daffadon/digihome/config/router"
	"github.com/daffadon/digihome/config/s3"
	"github.com/daffadon/digihome/internal/domain/handler"
	rag_repository "github.com/daffadon/digihome/internal/domain/repository/rag"
	"github.com/daffadon/digihome/internal/domain/services"
	s3_infra "github.com/daffadon/digihome/internal/infra/s3"
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
			s3.NewRustfsConnection,
			s3_infra.NewRustfsInfra,
			db.NewPostgresqlConn,
			rag_repository.NewRAGQueries,
			services.NewRagService,
			handler.NewRootHandler,
			handler.NewRagHandler,
			router.NewHTTPChi,
		),
		fx.Invoke(
			handler.RegisterRootRoute,
			handler.RegisterRagRoute,
			server.RunHTTPServer,
		),
	).Run()
}
