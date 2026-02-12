package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kiry163/filehub/internal/db"
)

const (
	MaxFolderDepth = 10
)

// 请求/响应结构体

type CreateFolderRequest struct {
	Name     string  `json:"name" binding:"required"`
	ParentID *string `json:"parent_id"`
}

type UpdateFolderRequest struct {
	Name string `json:"name" binding:"required"`
}

type MoveFolderRequest struct {
	ParentID *string `json:"parent_id"`
}

type MoveFileRequest struct {
	FolderID *string `json:"folder_id"`
}

type FolderResponse struct {
	FolderID  string  `json:"folder_id"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	ItemCount int     `json:"item_count,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type FolderContentsResponse struct {
	FolderID    string           `json:"folder_id"`
	Name        string           `json:"name"`
	ParentID    *string          `json:"parent_id"`
	Folders     []FolderResponse `json:"folders"`
	Files       []gin.H          `json:"files"`
	Stats       FolderStats      `json:"stats"`
	Breadcrumbs []BreadcrumbItem `json:"breadcrumbs"`
}

type FolderStats struct {
	FolderCount int   `json:"folder_count"`
	FileCount   int   `json:"file_count"`
	TotalSize   int64 `json:"total_size"`
}

type BreadcrumbItem struct {
	FolderID *string `json:"folder_id"`
	Name     string  `json:"name"`
}

// validationError
type validationError struct {
	field   string
	message string
}

func (e validationError) Error() string {
	return e.message
}

func validateFolderName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return validationError{field: "name", message: "folder name is required"}
	}
	if len(name) > 255 {
		return validationError{field: "name", message: "folder name too long (max 255)"}
	}

	invalidChars := `/\:*?"<>|`
	for _, char := range invalidChars {
		if strings.ContainsRune(name, char) {
			return validationError{field: "name", message: "folder name contains invalid characters"}
		}
	}
	return nil
}

// generateFolderID 生成文件夹ID（使用真正的随机字符串）
func generateFolderID() string {
	// 生成16 字节随机数据
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Printf("generateFolderID: crypto/rand.Read failed: %v", err)
		// 降级使用时间戳（概率极低冲突）
		return time.Now().Format("20060102") + "_" + generateRandomString(12)
	}
	// Base64 编码并移除特殊字符
	return strings.TrimRight(base64.URLEncoding.EncodeToString(randomBytes), "=")
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(length int) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Printf("generateRandomString: crypto/rand.Read failed: %v", err)
		return fmt.Sprintf("error-%d", time.Now().Unix())
	}
	result := make([]byte, length)
	for i := range result {
		result[i] = alphabet[int(randomBytes[i])%len(alphabet)]
	}
	return string(result)
}

// generateRandomString12 生成12位随机ID
func generateRandomString12() string {
	return generateRandomString(12)
}

// ==================== 文件夹管理 ====================

// CreateFolder 创建文件夹
func (h *Handler) CreateFolder(c *gin.Context) {
	var req CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.audit(c, "create_folder", "", getUser(c), "failure", "invalid request")
		Error(c, http.StatusBadRequest, 10004, "invalid request")
		return
	}

	// 验证名称
	if err := validateFolderName(req.Name); err != nil {
		h.audit(c, "create_folder", "", getUser(c), "failure", err.Error())
		Error(c, http.StatusBadRequest, 10004, err.Error())
		return
	}

	// 检查父文件夹是否存在（如果指定了）
	if req.ParentID != nil {
		log.Printf("[CreateFolder] 检查父文件夹: %s", *req.ParentID)
		_, err := h.Service.DB.GetFolder(c.Request.Context(), *req.ParentID)
		if err != nil {
			log.Printf("[CreateFolder] 父文件夹不存在: %v", *req.ParentID)
			h.audit(c, "create_folder", "", getUser(c), "failure", "parent folder not found")
			Error(c, http.StatusNotFound, 10003, "parent folder not found")
			return
		}
		log.Printf("[CreateFolder] 父文件夹验证通过: %v", *req.ParentID)
	}

	// 检查深度限制
	var depth int
	var err error
	if req.ParentID != nil {
		log.Printf("[CreateFolder] 检查深度: parent_id=%s", *req.ParentID)
		depth, err = h.Service.DB.GetFolderDepth(c.Request.Context(), *req.ParentID)
		if err != nil {
			log.Printf("[CreateFolder] 深度检查失败: %v", err)
			h.audit(c, "create_folder", "", getUser(c), "failure", "depth check failed")
			Error(c, http.StatusInternalServerError, 19999, "depth check failed")
			return
		}
	} else {
		log.Printf("[CreateFolder] 在根目录创建，深度: 0")
		depth = 0
	}
	log.Printf("[CreateFolder] 当前深度: %d, 创建后深度: %d", depth, depth+1)
	if depth+1 >= MaxFolderDepth {
		log.Printf("[CreateFolder] 深度超过限制: %d >= %d", depth+1, MaxFolderDepth)
		h.audit(c, "create_folder", "", getUser(c), "failure", "max depth exceeded")
		Error(c, http.StatusBadRequest, 10012, "max folder depth exceeded (10)")
		return
	}

	// 检查同名文件夹
	log.Printf("[CreateFolder] 检查同名: name=%s, parent_id=%v", req.Name, req.ParentID)
	_, err = h.Service.DB.GetFolderByName(c.Request.Context(), req.Name, req.ParentID)
	if err == nil {
		log.Printf("[CreateFolder] 文件夹已存在: name=%s, parent_id=%v", req.Name, req.ParentID)
		h.audit(c, "create_folder", "", getUser(c), "failure", "folder already exists")
		Error(c, http.StatusConflict, 10010, "folder already exists")
		return
	}
	log.Printf("[CreateFolder] 同名检查通过: %s", req.Name)

	// 创建文件夹
	now := time.Now().UTC().Format(time.RFC3339)
	log.Printf("[CreateFolder] 准备创建文件夹: folder_id=%s, name=%s, parent_id=%v", "", req.Name, req.ParentID)
	record := db.FolderRecord{
		FolderID:  generateFolderID(),
		Name:      req.Name,
		ParentID:  req.ParentID,
		CreatedBy: getUser(c),
		CreatedAt: now,
		UpdatedAt: now,
	}

	log.Printf("[CreateFolder] 开始数据库插入")
	if err := h.Service.DB.CreateFolder(c.Request.Context(), record); err != nil {
		log.Printf("[CreateFolder] 数据库插入失败: %v", err)
		h.audit(c, "create_folder", "", getUser(c), "failure", "database error")
		Error(c, http.StatusInternalServerError, 19999, "create folder failed")
		return
	}
	log.Printf("[CreateFolder] 数据库插入成功: %s", record.FolderID)

	h.audit(c, "create_folder", "", getUser(c), "success", "")
	OK(c, FolderResponse{
		FolderID:  record.FolderID,
		Name:      record.Name,
		ParentID:  record.ParentID,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	})
}

// ListFolders 列出文件夹
func (h *Handler) ListFolders(c *gin.Context) {
	parentID := c.Query("parent_id")
	var parentIDPtr *string
	if parentID != "" {
		parentIDPtr = &parentID
	}

	log.Printf("[ListFolders] 查询文件夹: parent_id=%v", parentIDPtr)
	records, err := h.Service.DB.ListFolders(c.Request.Context(), parentIDPtr)
	if err != nil {
		log.Printf("[ListFolders] 查询失败: %v", err)
		Error(c, http.StatusInternalServerError, 19999, "list folders failed")
		return
	}
	log.Printf("[ListFolders] 找到 %d 个文件夹", len(records))

	folders := make([]FolderResponse, 0, len(records))
	for _, record := range records {
		itemCount, _ := h.Service.DB.GetFolderItemCount(c.Request.Context(), record.FolderID)
		folders = append(folders, FolderResponse{
			FolderID:  record.FolderID,
			Name:      record.Name,
			ParentID:  record.ParentID,
			ItemCount: itemCount,
			CreatedAt: record.CreatedAt,
			UpdatedAt: record.UpdatedAt,
		})
	}
	OK(c, gin.H{"folders": folders})
}

// GetFolderContents 获取文件夹内容
func (h *Handler) GetFolderContents(c *gin.Context) {
	folderID := c.Param("id")

	log.Printf("[GetFolderContents] 获取文件夹内容: %s", folderID)

	// 获取文件夹信息
	folder, err := h.Service.DB.GetFolder(c.Request.Context(), folderID)
	if err != nil {
		log.Printf("[GetFolderContents] 文件夹不存在: %s: %v", folderID, err)
		Error(c, http.StatusNotFound, 10003, "folder not found")
		return
	}

	// 获取子文件夹
	folders, err := h.Service.DB.ListFolders(c.Request.Context(), &folderID)
	if err != nil {
		log.Printf("[GetFolderContents] 获取子文件夹失败: %s: %v", folderID, err)
		Error(c, http.StatusInternalServerError, 19999, "list folders failed")
		return
	}

	// 获取文件
	files, _, err := h.Service.DB.ListFilesByFolder(c.Request.Context(), &folderID, 1000, 0, "desc", "")
	if err != nil {
		log.Printf("[GetFolderContents] 获取文件失败: %s: %v", folderID, err)
		Error(c, http.StatusInternalServerError, 19999, "list files failed")
		return
	}

	// 获取统计数据
	folderCount := len(folders)
	fileCount, totalSize, _ := h.Service.DB.GetFolderStats(c.Request.Context(), folderID)

	// 构建面包屑
	breadcrumbs := h.buildBreadcrumbs(c.Request.Context(), folder)

	// 构建文件响应
	fileResponses := make([]gin.H, 0, len(files))
	for _, file := range files {
		fileResponses = append(fileResponses, gin.H{
			"file_id":       file.FileID,
			"original_name": file.OriginalName,
			"size":          file.Size,
			"mime_type":     file.MimeType,
			"filehub_url":   "filehub://" + file.FileID,
			"created_at":    file.CreatedAt,
			"download_url":  h.buildDownloadURL(c, file.FileID),
			"view_url":      h.buildBaseURL(c) + "/app/files/" + file.FileID,
		})
	}

	// 构建文件夹响应
	folderResponses := make([]FolderResponse, 0, len(folders))
	for _, folder := range folders {
		itemCount, _ := h.Service.DB.GetFolderItemCount(c.Request.Context(), folder.FolderID)
		folderResponses = append(folderResponses, FolderResponse{
			FolderID:  folder.FolderID,
			Name:      folder.Name,
			ParentID:  folder.ParentID,
			ItemCount: itemCount,
			CreatedAt: folder.CreatedAt,
			UpdatedAt: folder.UpdatedAt,
		})
	}

	OK(c, FolderContentsResponse{
		FolderID:    folder.FolderID,
		Name:        folder.Name,
		ParentID:    folder.ParentID,
		Folders:     folderResponses,
		Files:       fileResponses,
		Stats:       FolderStats{FolderCount: folderCount, FileCount: fileCount, TotalSize: totalSize},
		Breadcrumbs: breadcrumbs,
	})
}

// buildBreadcrumbs 构建面包屑导航
func (h *Handler) buildBreadcrumbs(ctx context.Context, folder db.FolderRecord) []BreadcrumbItem {
	items := []BreadcrumbItem{{FolderID: nil, Name: "Root"}}

	if folder.ParentID == nil {
		return items
	}

	// 递归向上收集路径
	currentID := folder.ParentID
	pathItems := []BreadcrumbItem{}
	for currentID != nil {
		currentFolder, err := h.Service.DB.GetFolder(ctx, *currentID)
		if err != nil {
			log.Printf("[buildBreadcrumbs] 获取父文件夹失败: %s, 错误: %v", *currentID, err)
			break
		}
		pathItems = append([]BreadcrumbItem{{FolderID: &currentFolder.FolderID, Name: currentFolder.Name}}, pathItems...)
		currentID = currentFolder.ParentID
	}

	return append(items, pathItems...)
}

// UpdateFolder 重命名文件夹
func (h *Handler) UpdateFolder(c *gin.Context) {
	folderID := c.Param("id")

	// 获取原文件夹
	folder, err := h.Service.DB.GetFolder(c.Request.Context(), folderID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "folder not found")
		return
	}

	// 验证名称
	if err := validateFolderName(c.PostForm("name")); err != nil {
		Error(c, http.StatusBadRequest, 10004, err.Error())
		return
	}

	newName := c.PostForm("name")
	if newName == folder.Name {
		h.audit(c, "rename_folder", folderID, getUser(c), "success", "")
		return
	}

	// 检查同名（排除自己）
	_, err = h.Service.DB.GetFolderByName(c.Request.Context(), newName, folder.ParentID)
	if err == nil {
		log.Printf("[UpdateFolder] 同名文件夹已存在: name=%s, parent_id=%v", newName, folder.ParentID)
		h.audit(c, "rename_folder", folderID, getUser(c), "failure", "folder already exists")
		Error(c, http.StatusConflict, 10010, "folder already exists")
		return
	}
	log.Printf("[UpdateFolder] 重命名文件夹: %s -> %s", folder.Name, newName)

	// 更新
	if err := h.Service.DB.UpdateFolder(c.Request.Context(), folderID, newName); err != nil {
		log.Printf("[UpdateFolder] 更新文件夹失败: %v", err)
		h.audit(c, "rename_folder", folderID, getUser(c), "failure", "rename failed")
		Error(c, http.StatusInternalServerError, 19999, "rename failed")
		return
	}

	// 返回响应
	h.audit(c, "rename_folder", folderID, getUser(c), "success", "")
	OK(c, gin.H{
		"folder_id":  folder.FolderID,
		"name":       newName,
		"parent_id":  folder.ParentID,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// MoveFolder 移动文件夹
func (h *Handler) MoveFolder(c *gin.Context) {
	folderID := c.Param("id")

	var req MoveFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, 10004, "invalid request")
		return
	}

	// 获取原文件夹
	folder, err := h.Service.DB.GetFolder(c.Request.Context(), folderID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "folder not found")
		return
	}

	// 不能移动到自己
	if *req.ParentID == folderID {
		Error(c, http.StatusBadRequest, 10013, "cannot move folder to itself")
		return
	}

	// 检查目标文件夹是否存在
	if req.ParentID != nil {
		_, err := h.Service.DB.GetFolder(c.Request.Context(), *req.ParentID)
		if err != nil {
			Error(c, http.StatusNotFound, 10003, "target folder not found")
			return
		}
	}

	// 检查循环引用（不能移动到自己内部）
	isDescendant, err := h.Service.DB.IsDescendant(c.Request.Context(), folderID, *req.ParentID)
	if err != nil || isDescendant {
		log.Printf("[MoveFolder] 检查循环引用失败: %v", err)
		Error(c, http.StatusBadRequest, 10013, "cannot move folder to its own subdirectory")
		return
	}

	// 检查深度限制
	log.Printf("[MoveFolder] 检查深度: current=%s, target=%s", folder.FolderID, *req.ParentID)
	targetDepth, _ := h.Service.DB.GetFolderDepth(c.Request.Context(), *req.ParentID)
	if err != nil {
		log.Printf("[MoveFolder] 目标文件夹深度检查失败: %v", err)
		Error(c, http.StatusInternalServerError, 19999, "depth check failed")
		return
	}
	log.Printf("[MoveFolder] 当前文件夹深度: %d, 移动后深度: %d", targetDepth)

	if targetDepth+1 >= MaxFolderDepth {
		Error(c, http.StatusBadRequest, 10012, "max folder depth exceeded (10)")
		return
	}

	// 检查同名（排除自己）
	_, err = h.Service.DB.GetFolderByName(c.Request.Context(), folder.Name, req.ParentID)
	if err == nil {
		log.Printf("[MoveFolder] 目标位置已存在同名文件夹: name=%s, parent_id=%v", folder.Name, req.ParentID)
		Error(c, http.StatusConflict, 10010, "folder already exists in target location")
		return
	}
	log.Printf("[MoveFolder] 同名检查通过")

	// 移动文件夹
	if err := h.Service.DB.MoveFolder(c.Request.Context(), folderID, req.ParentID); err != nil {
		log.Printf("[MoveFolder] 移动文件夹失败: %v", err)
		h.audit(c, "move_folder", folderID, getUser(c), "failure", "move folder failed")
		Error(c, http.StatusInternalServerError, 19999, "move folder failed")
		return
	}
	log.Printf("[MoveFolder] 移动文件夹成功: %s -> %s", folderID, *req.ParentID)
	h.audit(c, "move_folder", folderID, getUser(c), "success", "")
	Message(c, "moved")
}

// DeleteFolder 删除文件夹
func (h *Handler) DeleteFolder(c *gin.Context) {
	folderID := c.Param("id")

	log.Printf("[DeleteFolder] 开始删除文件夹: %s", folderID)

	// 检查是否为空
	itemCount, err := h.Service.DB.GetFolderItemCount(c.Request.Context(), folderID)
	if err != nil {
		log.Printf("[DeleteFolder] 检查文件夹内容失败: %s", err)
		Error(c, http.StatusInternalServerError, 19999, "check folder contents failed")
		return
	}
	log.Printf("[DeleteFolder] 文件夹内容检查完成: %s 个项目", itemCount)

	if itemCount > 0 {
		log.Printf("[DeleteFolder] 文件夹非空: %s 个项目", itemCount)
		h.audit(c, "delete_folder", folderID, getUser(c), "failure", "folder not empty")
		Error(c, http.StatusConflict, 10011, "folder is not empty")
		return
	}

	// 删除文件夹
	if err := h.Service.DB.DeleteFolder(c.Request.Context(), folderID); err != nil {
		log.Printf("[DeleteFolder] 删除文件夹失败: %v", err)
		h.audit(c, "delete_folder", folderID, getUser(c), "failure", "delete failed")
		Error(c, http.StatusInternalServerError, 19999, "delete failed")
		return
	}
	log.Printf("[DeleteFolder] 删除文件夹成功: %s", folderID)
	h.audit(c, "delete_folder", folderID, getUser(c), "success", "")
	Message(c, "deleted")
}

// GetFolderViewURL 获取文件夹访问链接
func (h *Handler) GetFolderViewURL(c *gin.Context) {
	folderID := c.Param("id")

	// 验证文件夹存在
	_, err := h.Service.DB.GetFolder(c.Request.Context(), folderID)
	if err != nil {
		log.Printf("[GetFolderViewURL] 文件夹不存在: %s: %v", folderID, err)
		Error(c, http.StatusNotFound, 10003, "folder not found")
		return
	}

	url := h.buildBaseURL(c) + "/app/folders/" + folderID
	OK(c, gin.H{"url": url})
}

// ==================== 文件移动相关 ====================

// MoveFile 移动文件到文件夹
func (h *Handler) MoveFile(c *gin.Context) {
	fileID := c.Param("id")

	var req MoveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, 10004, "invalid request")
		return
	}

	// 获取文件
	file, err := h.Service.DB.GetFileWithFolder(c.Request.Context(), fileID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "file not found")
		return
	}

	// 检查目标文件夹是否存在（如果指定了）
	if req.FolderID != nil {
		_, err := h.Service.DB.GetFolder(c.Request.Context(), *req.FolderID)
		if err != nil {
			log.Printf("[MoveFile] 目标文件夹不存在: %s", *req.FolderID, err)
			Error(c, http.StatusNotFound, 10003, "target folder not found")
			return
		}
		log.Printf("[MoveFile] 目标文件夹验证通过: %s", *req.FolderID)
	}

	// 检查同名文件
	files, _, err := h.Service.DB.ListFilesByFolder(c.Request.Context(), req.FolderID, 1000, 0, "desc", "")
	if err != nil {
		log.Printf("[MoveFile] 检查同名文件失败: %v", err)
		Error(c, http.StatusInternalServerError, 19999, "check files failed")
		return
	}
	log.Printf("[MoveFile] 开始检查同名文件，当前文件夹: %s, 共有 %d 个文件", req.FolderID, len(files))

	for _, f := range files {
		if f.OriginalName == file.OriginalName && f.FileID != fileID {
			log.Printf("[MoveFile] 目标文件夹已存在同名文件: name=%s", f.OriginalName)
			Error(c, http.StatusConflict, 10010, "file already exists in target folder")
			return
		}
	}

	log.Printf("[MoveFile] 同名文件检查通过，开始移动")

	// 移动文件
	if err := h.Service.DB.UpdateFileFolder(c.Request.Context(), fileID, req.FolderID); err != nil {
		log.Printf("[MoveFile] 移动文件失败: %v", err)
		h.audit(c, "move_file", fileID, getUser(c), "failure", "move file failed")
		Error(c, http.StatusInternalServerError, 19999, "move file failed")
		return
	}
	log.Printf("[MoveFile] 移动文件成功: %s -> %s", fileID, *req.FolderID)

	h.audit(c, "move_file", fileID, getUser(c), "success", "")
	Message(c, "moved")
}

// GetFileViewURL 获取文件访问页面链接
func (h *Handler) GetFileViewURL(c *gin.Context) {
	fileID := c.Param("id")

	// 验证文件存在
	_, err := h.Service.DB.GetFile(c.Request.Context(), fileID)
	if err != nil {
		Error(c, http.StatusNotFound, 10003, "file not found")
		return
	}

	url := h.buildBaseURL(c) + "/app/files/" + fileID
	OK(c, gin.H{"url": url})
}
