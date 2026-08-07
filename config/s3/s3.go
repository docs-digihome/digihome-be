package s3

import (
	"context"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

func NewRustfsConnection(slog *slog.Logger) *minio.Client {
	hostPort := viper.GetString("rustfs.host") + ":" + viper.GetString("rustfs.port")
	rustfsClient, err := minio.New(hostPort, &minio.Options{
		Creds: credentials.NewStaticV4(
			viper.GetString("rustfs.credential.user"),
			viper.GetString("rustfs.credential.password"),
			"",
		),
		Secure: false,
	})
	if err != nil {
		slog.Error("rustfs connection instance error", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = rustfsClient.ListBuckets(ctx)
	if err != nil {
		slog.Error("rustfs ping failed", "error", err)
	}

	return rustfsClient
}
