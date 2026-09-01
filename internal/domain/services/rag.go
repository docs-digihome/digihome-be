package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"os"
	"runtime"
	"slices"
	"sync"

	"github.com/daffadon/digihome/internal/constant"
	rag_repository "github.com/daffadon/digihome/internal/domain/repository/rag"
	s3_repository "github.com/daffadon/digihome/internal/domain/repository/s3"
	"github.com/daffadon/digihome/internal/pkg"
	"github.com/daffadon/digihome/internal/schema"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/pgvector/pgvector-go"
)

var ErrNoValidPDF = errors.New("no valid pdf files")

type (
	RagService interface {
		Seed(ctx context.Context) error
		DataPopulation(ctx context.Context, localPath, objectKey string) error
		BatchInsertDocument(ctx context.Context, files []*multipart.FileHeader) ([]schema.BatchInsertDocumentResponse, error)
		GetDocuments(ctx context.Context) ([]schema.GetSeededUniqueDocumentName, error)
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
	registeredDocumentsList := make([]string, len(registedDocuments))
	for _, val := range registedDocuments {
		registeredDocumentsList = append(registeredDocumentsList, val.DocumentName)
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
		constant.DOCUMENT_BUCKET,
		minio.ListObjectsOptions{
			Recursive: true,
		},
	)
	for object := range objects {
		if slices.Contains(registeredDocumentsList, object.Key) {
			continue
		}
		if object.Err != nil {
			r.slog.Error("list object failed",
				"error", object.Err,
			)
			continue
		}
		pdfPath, err := r.sr.CopyObjectToFile(ctx, constant.DOCUMENT_BUCKET, object.Key, pdfTempDir, "*.pdf")
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

// BatchInsertDocument implements [RagService].
func (r *ragService) BatchInsertDocument(ctx context.Context, files []*multipart.FileHeader) ([]schema.BatchInsertDocumentResponse, error) {
	workers := 4
	fileJobs := make(chan *multipart.FileHeader, len(files))
	results := make(chan schema.BatchInsertDocumentResponse, len(files))
	var wg sync.WaitGroup

	for range workers {
		wg.Go(func() {
			for file := range fileJobs {
				results <- r.insertDocument(ctx, file)
			}
		})
	}

	for _, file := range files {
		if file == nil {
			continue
		}
		fileJobs <- file
	}
	close(fileJobs)
	wg.Wait()
	close(results)

	responses := make([]schema.BatchInsertDocumentResponse, 0, len(files))
	for resp := range results {
		responses = append(responses, resp)
	}

	for _, resp := range responses {
		if resp.Error == "" {
			return responses, nil
		}
	}
	return responses, ErrNoValidPDF
}

func (r *ragService) insertDocument(ctx context.Context, file *multipart.FileHeader) schema.BatchInsertDocumentResponse {
	f, err := file.Open()
	if err != nil {
		r.slog.Error("open multipart file failed",
			"error", err,
			"name", file.Filename,
		)
		return schema.BatchInsertDocumentResponse{
			OriginalName: file.Filename,
			Error:        "failed to open file",
		}
	}

	isPdf, err := pkg.IsPDF(file.Filename, f)
	if err != nil {
		f.Close()
		r.slog.Error("pdf detection failed",
			"error", err,
			"name", file.Filename,
		)
		return schema.BatchInsertDocumentResponse{
			OriginalName: file.Filename,
			Error:        "failed to inspect file",
		}
	}
	if !isPdf {
		f.Close()
		r.slog.Warn("rejected non-pdf file",
			"name", file.Filename,
		)
		return schema.BatchInsertDocumentResponse{
			OriginalName: file.Filename,
			Error:        "not a pdf file",
		}
	}

	objectKey := uuid.NewString() + "_" + file.Filename
	if err := r.sr.Set(ctx, constant.DOCUMENT_BUCKET, objectKey, f, file.Size); err != nil {
		f.Close()
		r.slog.Error("set object failed",
			"error", err,
			"object-key", objectKey,
		)
		return schema.BatchInsertDocumentResponse{
			ObjectKey:    objectKey,
			OriginalName: file.Filename,
			Error:        err.Error(),
		}
	}
	f.Close()
	return schema.BatchInsertDocumentResponse{
		ObjectKey:    objectKey,
		OriginalName: file.Filename,
	}
}

// GetDocuments implements [RagService].
func (r *ragService) GetDocuments(ctx context.Context) ([]schema.GetSeededUniqueDocumentName, error) {
	docs, err := r.rr.GetRegisteredDocuments(ctx)
	if err != nil {
		return nil, err
	}
	if docs == nil {
		return nil, nil
	}
	res := make([]schema.GetSeededUniqueDocumentName, len(docs))
	for _, val := range docs {
		res = append(res, schema.GetSeededUniqueDocumentName{
			DocumentName: val.DocumentName,
			DocumentLink: val.Link,
		})
	}
	fmt.Println(res)
	return res, nil
}

// DataPopulation implements [RagService].
func (r *ragService) DataPopulation(ctx context.Context, localPath, objectKey string) error {
	defer os.Remove(localPath)
	result, err := pkg.InspectPDFContext(ctx, localPath)
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
		embedding, err := pkg.Embed(ctx, pkg.EmbedModel(), chunk.Content)
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
			Link:         constant.DOCUMENT_BUCKET + "/" + objectKey,
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
