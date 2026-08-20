package s3_repository

import (
	"context"
	"io"
	"os"

	s3_infra "github.com/daffadon/digihome/internal/infra/s3"
	"github.com/minio/minio-go/v7"
)

type (
	S3Repository interface {
		CopyObjectToFile(ctx context.Context, bucketName, objectPath, destDir, fileName string) (string, error)
		ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
		Set(ctx context.Context, bucketName, objectPath string, reader io.Reader, objectSize int64) error
	}
	s3Repository struct {
		rfi s3_infra.RustfsInfra
	}
)

func NewS3Repository(rfi s3_infra.RustfsInfra) S3Repository {
	return &s3Repository{
		rfi: rfi,
	}
}

// Set implements [S3Repository].
func (s *s3Repository) Set(ctx context.Context, bucketName string, objectPath string, reader io.Reader, objectSize int64) error {
	return s.rfi.Set(ctx, bucketName, objectPath, reader, objectSize)
}

// ListObjects implements [S3Repository].
func (s *s3Repository) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	return s.rfi.ListObjects(ctx, bucketName, opts)
}

// CopyObjectToFile implements [S3Repository].
func (s *s3Repository) CopyObjectToFile(ctx context.Context, bucketName string, objectPath string, destDir string, fileName string) (string, error) {
	obj, err := s.rfi.Get(
		ctx,
		bucketName,
		objectPath,
	)
	if err != nil {
		return "", err
	}
	defer obj.Close()

	os.MkdirAll(destDir, 0755)

	file, err := os.CreateTemp(destDir, fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, obj)
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}
