package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	sql *sql.DB
}

type FileRecord struct {
	FileID       string
	OriginalName string
	ObjectKey    string
	Size         int64
	MimeType     string
	CreatedBy    string
	CreatedAt    string
	UpdatedAt    string
}

type RefreshToken struct {
	Token     string
	ExpiresAt string
	IsRevoked bool
}

type ShareLink struct {
	Token     string
	FileID    string
	ExpiresAt string
	CreatedAt string
	CreatedBy string
	Status    string
}

func Open(path string) (*DB, error) {
	handle, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	handle.SetMaxOpenConns(1)
	db := &DB{sql: handle}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS files (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      file_id VARCHAR(12) UNIQUE NOT NULL,
      original_name VARCHAR(255) NOT NULL,
      object_key VARCHAR(512) NOT NULL,
      size BIGINT NOT NULL,
      mime_type VARCHAR(100),
      created_by VARCHAR(64) NOT NULL,
      created_at DATETIME NOT NULL,
      updated_at DATETIME NOT NULL,
      metadata JSON
    );`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      token VARCHAR(128) UNIQUE NOT NULL,
      expires_at DATETIME NOT NULL,
      is_revoked BOOLEAN DEFAULT FALSE,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      action VARCHAR(50) NOT NULL,
      file_id VARCHAR(32),
      actor VARCHAR(64) NOT NULL,
      ip_address VARCHAR(45),
      status VARCHAR(20),
      message TEXT,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`,
		`CREATE TABLE IF NOT EXISTS share_links (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      token VARCHAR(64) UNIQUE NOT NULL,
      file_id VARCHAR(32) NOT NULL,
      expires_at DATETIME NOT NULL,
      created_at DATETIME NOT NULL,
      created_by VARCHAR(64) NOT NULL,
      status VARCHAR(20) NOT NULL
    );`,
		`CREATE INDEX IF NOT EXISTS idx_share_links_file_id ON share_links(file_id);`,
		`CREATE INDEX IF NOT EXISTS idx_share_links_token ON share_links(token);`,
		`CREATE INDEX IF NOT EXISTS idx_files_created_by ON files(created_by);`,
		`CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at);`,
	}

	for _, stmt := range statements {
		if _, err := db.sql.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) CreateFile(ctx context.Context, record FileRecord) error {
	_, err := db.sql.ExecContext(
		ctx,
		`INSERT INTO files (file_id, original_name, object_key, size, mime_type, created_by, created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		record.FileID,
		record.OriginalName,
		record.ObjectKey,
		record.Size,
		record.MimeType,
		record.CreatedBy,
		record.CreatedAt,
		record.UpdatedAt,
	)
	return err
}

func (db *DB) GetFile(ctx context.Context, fileID string) (FileRecord, error) {
	var record FileRecord
	row := db.sql.QueryRowContext(ctx, `
    SELECT file_id, original_name, object_key, size, mime_type, created_by, created_at, updated_at
    FROM files WHERE file_id = ?`, fileID)
	if err := row.Scan(
		&record.FileID,
		&record.OriginalName,
		&record.ObjectKey,
		&record.Size,
		&record.MimeType,
		&record.CreatedBy,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return FileRecord{}, err
	}
	return record, nil
}

func (db *DB) ListFiles(ctx context.Context, limit, offset int, order, keyword string) ([]FileRecord, int, error) {
	if order != "asc" {
		order = "desc"
	}
	var total int
	countQuery := "SELECT COUNT(1) FROM files"
	args := []interface{}{}
	if keyword != "" {
		countQuery += " WHERE original_name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	if err := db.sql.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
    SELECT file_id, original_name, object_key, size, mime_type, created_by, created_at, updated_at
    FROM files`
	args = []interface{}{}
	if keyword != "" {
		query += " WHERE original_name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	query += fmt.Sprintf(" ORDER BY created_at %s LIMIT ? OFFSET ?", order)
	args = append(args, limit, offset)
	rows, err := db.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	records := make([]FileRecord, 0)
	for rows.Next() {
		var record FileRecord
		if err := rows.Scan(
			&record.FileID,
			&record.OriginalName,
			&record.ObjectKey,
			&record.Size,
			&record.MimeType,
			&record.CreatedBy,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		records = append(records, record)
	}
	return records, total, nil
}

func (db *DB) AddAuditLog(ctx context.Context, action, fileID, actor, ipAddress, status, message string) error {
	_, err := db.sql.ExecContext(
		ctx,
		`INSERT INTO audit_logs (action, file_id, actor, ip_address, status, message) VALUES (?, ?, ?, ?, ?, ?)`,
		action,
		fileID,
		actor,
		ipAddress,
		status,
		message,
	)
	return err
}

func (db *DB) DeleteFile(ctx context.Context, fileID string) (FileRecord, error) {
	record, err := db.GetFile(ctx, fileID)
	if err != nil {
		return FileRecord{}, err
	}
	_, err = db.sql.ExecContext(ctx, `DELETE FROM files WHERE file_id = ?`, fileID)
	if err != nil {
		return FileRecord{}, err
	}
	return record, nil
}

func (db *DB) CreateRefreshToken(ctx context.Context, token string, expiresAt string) error {
	_, err := db.sql.ExecContext(
		ctx,
		`INSERT INTO refresh_tokens (token, expires_at, is_revoked) VALUES (?, ?, false)`,
		token,
		expiresAt,
	)
	return err
}

func (db *DB) GetRefreshToken(ctx context.Context, token string) (RefreshToken, error) {
	var record RefreshToken
	row := db.sql.QueryRowContext(ctx, `
    SELECT token, expires_at, is_revoked
    FROM refresh_tokens WHERE token = ?`, token)
	if err := row.Scan(&record.Token, &record.ExpiresAt, &record.IsRevoked); err != nil {
		return RefreshToken{}, err
	}
	return record, nil
}

func (db *DB) RevokeRefreshToken(ctx context.Context, token string) error {
	_, err := db.sql.ExecContext(ctx, `UPDATE refresh_tokens SET is_revoked = true WHERE token = ?`, token)
	return err
}

func (db *DB) RevokeAllRefreshTokens(ctx context.Context) error {
	_, err := db.sql.ExecContext(ctx, `UPDATE refresh_tokens SET is_revoked = true`)
	return err
}

func (db *DB) CreateShareLink(ctx context.Context, link ShareLink) error {
	_, err := db.sql.ExecContext(
		ctx,
		`INSERT INTO share_links (token, file_id, expires_at, created_at, created_by, status)
	 VALUES (?, ?, ?, ?, ?, ?)`,
		link.Token,
		link.FileID,
		link.ExpiresAt,
		link.CreatedAt,
		link.CreatedBy,
		link.Status,
	)
	return err
}

func (db *DB) GetShareLink(ctx context.Context, token string) (ShareLink, error) {
	var link ShareLink
	row := db.sql.QueryRowContext(ctx, `
    SELECT token, file_id, expires_at, created_at, created_by, status
    FROM share_links WHERE token = ?`, token)
	if err := row.Scan(
		&link.Token,
		&link.FileID,
		&link.ExpiresAt,
		&link.CreatedAt,
		&link.CreatedBy,
		&link.Status,
	); err != nil {
		return ShareLink{}, err
	}
	return link, nil
}

func (db *DB) GetActiveShareLink(ctx context.Context, fileID, nowRFC3339 string) (ShareLink, error) {
	var link ShareLink
	row := db.sql.QueryRowContext(ctx, `
    SELECT token, file_id, expires_at, created_at, created_by, status
    FROM share_links
    WHERE file_id = ? AND status = 'active' AND expires_at > ?
    ORDER BY created_at DESC
    LIMIT 1`, fileID, nowRFC3339)
	if err := row.Scan(
		&link.Token,
		&link.FileID,
		&link.ExpiresAt,
		&link.CreatedAt,
		&link.CreatedBy,
		&link.Status,
	); err != nil {
		return ShareLink{}, err
	}
	return link, nil
}

func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
