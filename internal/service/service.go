package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"mime/multipart"
	"time"

	"github.com/kiry163/filehub/internal/config"
	"github.com/kiry163/filehub/internal/db"
	"github.com/kiry163/filehub/internal/storage"
)

type Service struct {
	DB      *db.DB
	Storage storage.Storage
	Config  config.Config
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (s *Service) Login(ctx context.Context, username, password string) (Tokens, error) {
	if username != s.Config.Auth.AdminUsername || password != s.Config.Auth.AdminPassword {
		return Tokens{}, errors.New("invalid credentials")
	}
	return s.issueTokens(ctx, username)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	record, err := s.DB.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return Tokens{}, err
	}
	if record.IsRevoked {
		return Tokens{}, errors.New("refresh token revoked")
	}
	expiresAt, err := time.Parse(time.RFC3339, record.ExpiresAt)
	if err != nil {
		return Tokens{}, err
	}
	if time.Now().UTC().After(expiresAt) {
		return Tokens{}, errors.New("refresh token expired")
	}
	_ = s.DB.RevokeRefreshToken(ctx, refreshToken)
	return s.issueTokens(ctx, s.Config.Auth.AdminUsername)
}

func (s *Service) Logout(ctx context.Context) error {
	return s.DB.RevokeAllRefreshTokens(ctx)
}

func (s *Service) Upload(ctx context.Context, header *multipart.FileHeader, createdBy string) (db.FileRecord, error) {
	reader, err := header.Open()
	if err != nil {
		return db.FileRecord{}, err
	}
	defer reader.Close()

	fileID := generateFileID(12)
	saveResult, err := s.Storage.Save(ctx, reader, header.Size, fileID, header.Filename)
	if err != nil {
		return db.FileRecord{}, err
	}

	now := db.NowRFC3339()
	record := db.FileRecord{
		FileID:       fileID,
		OriginalName: header.Filename,
		ObjectKey:    saveResult.ObjectKey,
		Size:         saveResult.Size,
		MimeType:     saveResult.MimeType,
		CreatedBy:    createdBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.DB.CreateFile(ctx, record); err != nil {
		return db.FileRecord{}, err
	}
	return record, nil
}

func (s *Service) GetFile(ctx context.Context, fileID string) (db.FileRecord, error) {
	return s.DB.GetFile(ctx, fileID)
}

func (s *Service) ListFiles(ctx context.Context, limit, offset int, order, keyword string) ([]db.FileRecord, int, error) {
	return s.DB.ListFiles(ctx, limit, offset, order, keyword)
}

func (s *Service) DeleteFile(ctx context.Context, fileID string) (db.FileRecord, error) {
	record, err := s.DB.DeleteFile(ctx, fileID)
	if err != nil {
		return db.FileRecord{}, err
	}
	_ = s.Storage.Delete(ctx, record.ObjectKey)
	return record, nil
}

func (s *Service) GetObject(ctx context.Context, objectKey string, rangeStart, rangeEnd *int64) (storage.ObjectInfo, io.ReadCloser, error) {
	reader, info, err := s.Storage.Get(ctx, objectKey, rangeStart, rangeEnd)
	if err != nil {
		return storage.ObjectInfo{}, nil, err
	}
	return info, reader, nil
}

func generateFileID(length int) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	buf := make([]byte, length)
	random := make([]byte, length)
	_, _ = rand.Read(random)
	for i := range buf {
		buf[i] = alphabet[int(random[i])%len(alphabet)]
	}
	return string(buf)
}

func (s *Service) issueTokens(ctx context.Context, username string) (Tokens, error) {
	accessToken, expiresIn, err := s.newAccessToken(username)
	if err != nil {
		return Tokens{}, err
	}

	refreshToken, err := randomToken(48)
	if err != nil {
		return Tokens{}, err
	}
	refreshExpiresAt := time.Now().UTC().Add(time.Hour * 24 * time.Duration(s.Config.Auth.RefreshExpireDays)).Format(time.RFC3339)
	if err := s.DB.CreateRefreshToken(ctx, refreshToken, refreshExpiresAt); err != nil {
		return Tokens{}, err
	}

	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

func randomToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
