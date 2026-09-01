package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/daffadon/digihome/internal/constant"
	s3_infra "github.com/daffadon/digihome/internal/infra/s3"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func RunHTTPServer(lc fx.Lifecycle, r chi.Router, slog *slog.Logger, s s3_infra.RustfsInfra) *http.Server {
	srv := &http.Server{Addr: ":" + viper.GetString("app.http.port"), Handler: r}
	err := s.InitBucket(context.Background(), constant.DOCUMENT_BUCKET)
	if err != nil {
		slog.Error("Failed to init bucket", "error", err)
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			slog.Info("Starting HTTP server at" + srv.Addr)
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
