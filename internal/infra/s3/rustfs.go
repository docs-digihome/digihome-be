package s3_infra

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/daffadon/digihome/internal/constant"
	"github.com/daffadon/digihome/internal/pkg"
	"github.com/minio/minio-go/v7"
)

type (
	RustfsInfra interface {
		InitBucket(context context.Context, bucketName string) error
		Set(ctx context.Context, bucketName, objectPath string, reader io.Reader, objectSize int64) error
		Get(ctx context.Context, bucketName, objectPath string) (io.ReadCloser, error)
		Delete(ctx context.Context, bucketName, objectPath string) error
		CreateBucketIfNotExist(ctx context.Context, bucketName string) error
		ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
		SetPolicy(ctx context.Context, bucketName, policy string) error
		GetPolicy(ctx context.Context, bucketName string) (string, error)
	}
	rustfsInfra struct {
		client *minio.Client
		logger *slog.Logger
	}
)

func NewRustfsInfra(client *minio.Client, logger *slog.Logger) RustfsInfra {
	return &rustfsInfra{
		client: client,
		logger: logger,
	}
}

// listObjects implements [RustfsInfra].
func (r *rustfsInfra) ListObjects(
	ctx context.Context,
	bucketName string,
	opts minio.ListObjectsOptions,
) <-chan minio.ObjectInfo {
	return r.client.ListObjects(ctx, bucketName, opts)
}

// InitBucket implements [RustfsInfra].
func (r *rustfsInfra) InitBucket(context context.Context, bucketName string) error {
	err := r.CreateBucketIfNotExist(context, bucketName)
	if err != nil {
		return err
	}
	policyStr, err := r.GetPolicy(context, bucketName)
	if err != nil {
		return err
	}
	if policyStr != "" {
		public, err := pkg.IsBucketPublic(policyStr)
		if err != nil {
			return err
		}
		if !public {
			policy := fmt.Sprintf(constant.PUBLIC_PERMISSION, bucketName)
			err = r.SetPolicy(context, bucketName, policy)
			if err != nil {
				return err
			}
		}
	} else {
		policy := fmt.Sprintf(constant.PUBLIC_PERMISSION, bucketName)
		err = r.SetPolicy(context, bucketName, policy)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *rustfsInfra) Set(ctx context.Context, bucketName, objectPath string, reader io.Reader, objectSize int64) error {
	_, err := r.client.PutObject(ctx, bucketName, objectPath, reader, objectSize, minio.PutObjectOptions{})
	if err != nil {
		r.logger.Error(
			"put object failed",
			"error", err,
			"bucket-name", bucketName,
			"object-path", objectPath,
		)
	}
	return err
}

func (r *rustfsInfra) Get(ctx context.Context, bucketName, objectPath string) (io.ReadCloser, error) {
	object, err := r.client.GetObject(ctx, bucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		r.logger.Error(
			"get object failed",
			"error", err,
			"bucket-name", bucketName,
			"object-path", objectPath,
		)
		return nil, err
	}
	return object, nil
}

func (r *rustfsInfra) Delete(ctx context.Context, bucketName, objectPath string) error {
	if err := r.client.RemoveObject(ctx, bucketName, objectPath, minio.RemoveObjectOptions{}); err != nil {
		r.logger.Error(
			"delete object failed",
			"error", err,
			"bucket-name", bucketName,
			"object-path", objectPath,
		)
		return err
	}
	return nil
}

func (r *rustfsInfra) CreateBucketIfNotExist(ctx context.Context, bucketName string) error {
	exists, err := r.client.BucketExists(ctx, bucketName)
	if err != nil {
		r.logger.Error(
			"check bucket name existance failed",
			"error", err,
			"bucket-name", bucketName,
		)
		return err
	}
	if !exists {
		if err := r.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			r.logger.Error(
				"make bucket failed",
				"error", err,
				"bucket-name", bucketName,
			)
			return err
		}
		return nil
	}
	return nil
}
func (r *rustfsInfra) SetPolicy(ctx context.Context, bucketName, policy string) error {
	if err := r.client.SetBucketPolicy(ctx, bucketName, policy); err != nil {
		r.logger.Error(
			"set bucket policy failed",
			"error", err,
			"bucket-name", bucketName,
		)
		return err
	}
	return nil
}

func (r *rustfsInfra) GetPolicy(ctx context.Context, bucketName string) (string, error) {
	pol, err := r.client.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		r.logger.Error(
			"get bucket policy failed",
			"error", err,
			"bucket-name", bucketName,
		)
		return "", err
	}
	return pol, nil
}
