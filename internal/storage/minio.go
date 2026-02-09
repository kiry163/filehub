package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/kiry163/filehub/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client *minio.Client
	bucket string
}

func NewMinioStorage(ctx context.Context, cfg config.MinioConfig) (*MinioStorage, error) {
	var client *minio.Client
	var err error

	// Retry connecting to MinIO with exponential backoff
	maxRetries := 10
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		client, err = minio.New(cfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.UseSSL,
			Region: cfg.Region,
		})
		if err == nil {
			// Test connection by checking bucket existence
			_, bucketErr := client.BucketExists(ctx, cfg.Bucket)
			if bucketErr == nil {
				break
			}
			err = bucketErr
		}

		if i < maxRetries-1 {
			log.Printf("Waiting for MinIO to be ready... (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(retryDelay)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to MinIO after %d attempts: %w", maxRetries, err)
	}

	if err := ensureBucket(ctx, client, cfg.Bucket); err != nil {
		return nil, err
	}
	return &MinioStorage{client: client, bucket: cfg.Bucket}, nil
}

func (s *MinioStorage) Save(ctx context.Context, reader io.Reader, size int64, fileID, originalName string) (SaveResult, error) {
	ext := strings.ToLower(filepath.Ext(originalName))
	if ext == "" {
		ext = ".bin"
	}
	objectKey := time.Now().UTC().Format("2006-01-02") + "/" + fileID + ext

	buf := make([]byte, 512)
	n, _ := io.ReadFull(reader, buf)
	mimeType := "application/octet-stream"
	if n > 0 {
		mimeType = http.DetectContentType(buf[:n])
	}
	contentReader := io.MultiReader(bytes.NewReader(buf[:n]), reader)
	_, err := s.client.PutObject(
		ctx,
		s.bucket,
		objectKey,
		contentReader,
		size,
		minio.PutObjectOptions{ContentType: mimeType},
	)
	if err != nil {
		return SaveResult{}, err
	}
	return SaveResult{ObjectKey: objectKey, Size: size, MimeType: mimeType}, nil
}

func (s *MinioStorage) Get(ctx context.Context, objectKey string, rangeStart, rangeEnd *int64) (io.ReadCloser, ObjectInfo, error) {
	stat, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	opts := minio.GetObjectOptions{}
	if rangeStart != nil && rangeEnd != nil {
		opts.SetRange(*rangeStart, *rangeEnd)
	}
	object, err := s.client.GetObject(ctx, s.bucket, objectKey, opts)
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	return object, ObjectInfo{Size: stat.Size, ContentType: stat.ContentType}, nil
}

func (s *MinioStorage) Stat(ctx context.Context, objectKey string) (ObjectInfo, error) {
	stat, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Size: stat.Size, ContentType: stat.ContentType}, nil
}

func (s *MinioStorage) Delete(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}
