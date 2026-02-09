package storage

import (
	"context"
	"io"
)

type SaveResult struct {
	ObjectKey string
	Size      int64
	MimeType  string
}

type ObjectInfo struct {
	Size        int64
	ContentType string
}

type Storage interface {
	Save(ctx context.Context, reader io.Reader, size int64, fileID, originalName string) (SaveResult, error)
	Get(ctx context.Context, objectKey string, rangeStart, rangeEnd *int64) (io.ReadCloser, ObjectInfo, error)
	Stat(ctx context.Context, objectKey string) (ObjectInfo, error)
	Delete(ctx context.Context, objectKey string) error
}
