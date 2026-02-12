package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kiry163/filehub/internal/db"
	"github.com/kiry163/filehub/internal/service"
)

type Handler struct {
	Service *service.Service
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Health(c *gin.Context) {
	OK(c, gin.H{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		Error(c, http.StatusBadRequest, 10004, "invalid request")
		h.audit(c, "login", "", req.Username, "failure", "invalid request")
		return
	}
	tokens, err := h.Service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		Error(c, http.StatusUnauthorized, 10008, "login failed")
		h.audit(c, "login", "", req.Username, "failure", "login failed")
		return
	}
	h.audit(c, "login", "", req.Username, "success", "")
	OK(c, tokens)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		Error(c, http.StatusBadRequest, 10004, "invalid request")
		h.audit(c, "refresh", "", "system", "failure", "invalid request")
		return
	}
	tokens, err := h.Service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		Error(c, http.StatusUnauthorized, 10009, "refresh token invalid")
		h.audit(c, "refresh", "", "system", "failure", "refresh failed")
		return
	}
	h.audit(c, "refresh", "", "system", "success", "")
	OK(c, tokens)
}

func (h *Handler) Logout(c *gin.Context) {
	if err := h.Service.Logout(c.Request.Context()); err != nil {
		Error(c, http.StatusInternalServerError, 19999, "logout failed")
		h.audit(c, "logout", "", getUser(c), "failure", "logout failed")
		return
	}
	h.audit(c, "logout", "", getUser(c), "success", "")
	Message(c, "logged_out")
}

func (h *Handler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Error(c, http.StatusBadRequest, 10004, "file required")
		h.audit(c, "upload", "", getUser(c), "failure", "file required")
		return
	}
	maxBytes := h.Service.Config.Upload.MaxSizeMB * 1024 * 1024
	if maxBytes > 0 && file.Size > maxBytes {
		Error(c, http.StatusBadRequest, 10004, "file too large")
		h.audit(c, "upload", "", getUser(c), "failure", "file too large")
		return
	}

	user := getUser(c)
	record, err := h.Service.Upload(c.Request.Context(), file, user)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, 10005, "upload failed")
		h.audit(c, "upload", "", user, "failure", "upload failed")
		return
	}
	h.audit(c, "upload", record.FileID, user, "success", "")
	OK(c, gin.H{
		"file_id":       record.FileID,
		"filehub_url":   "filehub://" + record.FileID,
		"original_name": record.OriginalName,
		"size":          record.Size,
		"created_at":    record.CreatedAt,
		"download_url":  h.buildDownloadURL(c, record.FileID),
	})
}

func (h *Handler) GetFile(c *gin.Context) {
	fileID := c.Param("id")
	record, err := h.Service.GetFile(c.Request.Context(), fileID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "not found")
		return
	}
	OK(c, gin.H{
		"file_id":       record.FileID,
		"original_name": record.OriginalName,
		"size":          record.Size,
		"mime_type":     record.MimeType,
		"filehub_url":   "filehub://" + record.FileID,
		"created_at":    record.CreatedAt,
		"download_url":  h.buildDownloadURL(c, record.FileID),
	})
}

func (h *Handler) ListFiles(c *gin.Context) {
	limit := parseInt(c.DefaultQuery("limit", "20"), 20)
	offset := parseInt(c.DefaultQuery("offset", "0"), 0)
	order := c.DefaultQuery("order", "desc")
	keyword := strings.TrimSpace(c.Query("keyword"))
	records, total, err := h.Service.ListFiles(c.Request.Context(), limit, offset, order, keyword)
	if err != nil {
		Error(c, http.StatusInternalServerError, 19999, "list failed")
		return
	}
	files := make([]gin.H, 0, len(records))
	for _, record := range records {
		files = append(files, gin.H{
			"file_id":       record.FileID,
			"original_name": record.OriginalName,
			"size":          record.Size,
			"filehub_url":   "filehub://" + record.FileID,
			"created_at":    record.CreatedAt,
			"download_url":  h.buildDownloadURL(c, record.FileID),
		})
	}
	OK(c, gin.H{"total": total, "files": files})
}

func (h *Handler) PreviewFile(c *gin.Context) {
	fileID := c.Param("id")
	actor, ok := h.authorize(c)
	if !ok {
		Error(c, http.StatusUnauthorized, 10001, "unauthorized")
		return
	}
	_, err := h.Service.GetFile(c.Request.Context(), fileID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "not found")
		h.audit(c, "preview", fileID, actor, "failure", "not found")
		return
	}
	token, err := h.Service.NewPreviewToken(fileID, 10*time.Minute)
	if err != nil {
		Error(c, http.StatusInternalServerError, 19999, "preview failed")
		h.audit(c, "preview", fileID, actor, "failure", "token failed")
		return
	}
	url := h.buildPreviewStreamURL(c, token)
	h.audit(c, "preview", fileID, actor, "success", "")
	OK(c, gin.H{"url": url})
}

func (h *Handler) StreamFile(c *gin.Context) {
	token := c.Query("token")
	claims, err := h.Service.ParsePreviewToken(token)
	if err != nil {
		Error(c, http.StatusUnauthorized, 10001, "unauthorized")
		return
	}
	fileID := claims.Subject
	record, err := h.Service.GetFile(c.Request.Context(), fileID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "not found")
		return
	}
	if err := h.streamObject(c, record, true); err != nil {
		Error(c, http.StatusInternalServerError, 10006, "download failed")
		return
	}
}

func (h *Handler) DeleteFile(c *gin.Context) {
	fileID := c.Param("id")
	_, err := h.Service.DeleteFile(c.Request.Context(), fileID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "not found")
		h.audit(c, "delete", fileID, getUser(c), "failure", "not found")
		return
	}
	h.audit(c, "delete", fileID, getUser(c), "success", "")
	Message(c, "deleted")
}

func (h *Handler) DownloadFile(c *gin.Context) {
	fileID := c.Param("id")
	record, err := h.Service.GetFile(c.Request.Context(), fileID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "not found")
		h.audit(c, "download", fileID, getUser(c), "failure", "not found")
		return
	}
	if err := h.streamObject(c, record, false); err != nil {
		Error(c, http.StatusInternalServerError, 10006, "download failed")
		h.audit(c, "download", fileID, getUser(c), "failure", "stream error")
		return
	}
	h.audit(c, "download", fileID, getUser(c), "success", "")
}

func (h *Handler) ShareFile(c *gin.Context) {
	fileID := c.Param("id")
	_, err := h.Service.GetFile(c.Request.Context(), fileID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "not found")
		h.audit(c, "share", fileID, getUser(c), "failure", "not found")
		return
	}

	now := time.Now().UTC()
	link, err := h.Service.DB.GetActiveShareLink(c.Request.Context(), fileID, now.Format(time.RFC3339))
	if err == nil {
		h.audit(c, "share", fileID, getUser(c), "success", "reused")
		OK(c, gin.H{"url": h.buildShareDownloadURL(c, link.Token), "expires_at": link.ExpiresAt})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		Error(c, http.StatusInternalServerError, 19999, "share failed")
		h.audit(c, "share", fileID, getUser(c), "failure", "query failed")
		return
	}

	expiresAt := now.Add(7 * 24 * time.Hour)
	link = db.ShareLink{
		Token:     generateShareToken(32),
		FileID:    fileID,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
		CreatedBy: getUser(c),
		Status:    "active",
	}
	if err := h.Service.DB.CreateShareLink(c.Request.Context(), link); err != nil {
		Error(c, http.StatusInternalServerError, 19999, "share failed")
		h.audit(c, "share", fileID, getUser(c), "failure", "create failed")
		return
	}
	h.audit(c, "share", fileID, getUser(c), "success", "created")
	OK(c, gin.H{"url": h.buildShareDownloadURL(c, link.Token), "expires_at": link.ExpiresAt})
}

func (h *Handler) DownloadShare(c *gin.Context) {
	token := c.Param("token")
	link, err := h.Service.DB.GetShareLink(c.Request.Context(), token)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	if link.Status != "active" {
		c.Status(http.StatusNotFound)
		return
	}
	expiresAt, err := time.Parse(time.RFC3339, link.ExpiresAt)
	if err != nil || time.Now().UTC().After(expiresAt) {
		c.Status(http.StatusNotFound)
		return
	}
	record, err := h.Service.GetFile(c.Request.Context(), link.FileID)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	if err := h.streamObject(c, record, false); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
}

func getUser(c *gin.Context) string {
	if value, ok := c.Get("user"); ok {
		if user, ok := value.(string); ok {
			return user
		}
	}
	return "admin"
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (h *Handler) buildDownloadURL(c *gin.Context, fileID string) string {
	return h.buildBaseURL(c) + "/api/v1/files/" + fileID + "/download"
}

func (h *Handler) buildShareDownloadURL(c *gin.Context, token string) string {
	return h.buildBaseURL(c) + "/s/" + token
}

func (h *Handler) buildPreviewStreamURL(c *gin.Context, token string) string {
	return h.buildBaseURL(c) + "/api/v1/files/stream?token=" + token
}

func generateShareToken(length int) string {
	buf := make([]byte, length)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func (h *Handler) buildBaseURL(c *gin.Context) string {
	if h.Service.Config.Server.PublicEndpoint != "" {
		return h.Service.Config.Server.PublicEndpoint
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := c.GetHeader("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	return scheme + "://" + c.Request.Host
}

func parseRangeHeader(value string, size int64) (*int64, *int64, bool, error) {
	if value == "" {
		return nil, nil, false, nil
	}
	if !strings.HasPrefix(value, "bytes=") {
		return nil, nil, false, errors.New("invalid range")
	}
	parts := strings.Split(strings.TrimPrefix(value, "bytes="), "-")
	if len(parts) != 2 {
		return nil, nil, false, errors.New("invalid range")
	}
	if parts[0] == "" {
		suffix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffix <= 0 {
			return nil, nil, false, errors.New("invalid range")
		}
		start := size - suffix
		end := size - 1
		if start < 0 {
			start = 0
		}
		return &start, &end, true, nil
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		return nil, nil, false, errors.New("invalid range")
	}
	var end int64
	if parts[1] == "" {
		end = size - 1
	} else {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return nil, nil, false, errors.New("invalid range")
		}
	}
	if start >= size {
		return nil, nil, false, errors.New("invalid range")
	}
	if end >= size {
		end = size - 1
	}
	return &start, &end, true, nil
}

func buildContentRange(start, end, size int64) string {
	return "bytes " + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10) + "/" + strconv.FormatInt(size, 10)
}

func (h *Handler) audit(c *gin.Context, action, fileID, actor, status, message string) {
	_ = h.Service.DB.AddAuditLog(c.Request.Context(), action, fileID, actor, c.ClientIP(), status, message)
}

func (h *Handler) authorize(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		claims, err := h.Service.ParseAccessToken(strings.TrimPrefix(authHeader, "Bearer "))
		if err == nil {
			return claims.Subject, true
		}
	}
	localKey := c.GetHeader("X-Local-Key")
	if localKey != "" && localKey == h.Service.Config.Auth.LocalKey {
		return "local", true
	}
	return "", false
}

func (h *Handler) streamObject(c *gin.Context, record db.FileRecord, inline bool) error {
	objectInfo, err := h.Service.Storage.Stat(c.Request.Context(), record.ObjectKey)
	if err != nil {
		return err
	}
	start, end, partial, err := parseRangeHeader(c.GetHeader("Range"), objectInfo.Size)
	if err != nil {
		return err
	}
	reader, _, err := h.Service.Storage.Get(c.Request.Context(), record.ObjectKey, start, end)
	if err != nil {
		return err
	}
	defer reader.Close()

	disposition := "attachment"
	if inline {
		disposition = "inline"
	}

	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", objectInfo.ContentType)
	c.Header("Content-Disposition", disposition+"; filename=\""+record.OriginalName+"\"")
	if partial {
		c.Status(http.StatusPartialContent)
		c.Header("Content-Range", buildContentRange(*start, *end, objectInfo.Size))
		c.Header("Content-Length", strconv.FormatInt(*end-*start+1, 10))
	} else {
		c.Status(http.StatusOK)
		c.Header("Content-Length", strconv.FormatInt(objectInfo.Size, 10))
	}
	_, err = io.Copy(c.Writer, reader)
	return err
}
