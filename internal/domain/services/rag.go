package services

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"sync"

	"github.com/daffadon/digihome/internal/constant"
	rag_repository "github.com/daffadon/digihome/internal/domain/repository/rag"
	s3_repository "github.com/daffadon/digihome/internal/domain/repository/s3"
	"github.com/daffadon/digihome/internal/pkg"
	"github.com/daffadon/digihome/internal/schema"
	"github.com/minio/minio-go/v7"
	"github.com/pgvector/pgvector-go"
)

type (
	RagService interface {
		Seed(ctx context.Context) error
		DataPopulation(ctx context.Context, localPath, objectKey string) error
	}
	ragService struct {
		slog       *slog.Logger
		rr         *rag_repository.Queries
		sr         s3_repository.S3Repository
		normalizer pkg.MarkDownNormalizer
		chunker    pkg.MarkdownChunker
	}
)

func NewRagService(slog *slog.Logger, rr *rag_repository.Queries, sr s3_repository.S3Repository, normalizer pkg.MarkDownNormalizer, chunker pkg.MarkdownChunker) RagService {
	return &ragService{
		slog:       slog,
		rr:         rr,
		sr:         sr,
		normalizer: normalizer,
		chunker:    chunker,
	}
}

// Seed implements [RagService].
func (r *ragService) Seed(ctx context.Context) error {
	workers := runtime.NumCPU()
	pdfJobs := make(chan schema.DataPopulationProps, workers*2)
	var wg sync.WaitGroup

	registedDocuments, err := r.rr.GetRegisteredDocuments(ctx)
	if err != nil {
		return err
	}

	for range workers {
		wg.Go(func() {
			for pdfJob := range pdfJobs {
				if err := r.DataPopulation(ctx, pdfJob.LocalPath, pdfJob.ObjectKey); err != nil {
					r.slog.Error("data population failed",
						"error", err,
						"path", pdfJob.ObjectKey,
					)
				}
			}
		})
	}

	pdfTempDir := "./temp/pdf-processing"
	objects := r.sr.ListObjects(
		ctx,
		"documents",
		minio.ListObjectsOptions{
			Recursive: true,
		},
	)
	for object := range objects {
		if slices.Contains(registedDocuments, object.Key) {
			continue
		}
		if object.Err != nil {
			r.slog.Error("list object failed",
				"error", object.Err,
			)
			continue
		}
		pdfPath, err := r.sr.CopyObjectToFile(ctx, "documents", object.Key, pdfTempDir, "*.pdf")
		if err != nil {
			r.slog.Error("copy file failed",
				"error", err,
				"path", object.Key,
			)
			continue
		}
		pdfJobs <- schema.DataPopulationProps{LocalPath: pdfPath, ObjectKey: object.Key}
	}

	close(pdfJobs)
	wg.Wait()
	return nil
}

// DataPopulation implements [RagService].
func (r *ragService) DataPopulation(ctx context.Context, localPath, objectKey string) error {
	defer os.Remove(localPath)
	result, err := pkg.InspectPDF(localPath)
	if err != nil {
		r.slog.Error("pdf inspect failed",
			"error", err,
			"path", objectKey,
		)
		return err
	}
	normalized := r.normalizer.Normalize(string(result))
	chunks := r.chunker.Split(normalized)
	for _, chunk := range chunks {
		if chunk.Content == "" {
			continue
		}
		embedding, err := pkg.Embed(ctx, constant.DEFAULT_EMBED_MODEL, chunk.Content)
		if err != nil {
			r.slog.Error("embedding process error",
				"error", err,
				"path", objectKey,
				"chunk-index", chunk.Index,
			)
			continue
		}
		vector := pgvector.NewVector(embedding)
		if err := r.rr.InsertDocumentChunks(ctx, rag_repository.InsertDocumentChunksParams{
			DocumentName: objectKey,
			ChunkIndex:   int32(chunk.Index),
			Content:      chunk.Content,
			Embedding:    vector,
		}); err != nil {
			r.slog.Error("insert chunk error",
				"error", err,
				"path", objectKey,
				"chunk-index", chunk.Index,
			)
			continue
		}
	}
	return nil
}
