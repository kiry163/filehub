package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	Endpoint string
	LocalKey string
	HTTP     *http.Client
}

type APIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type FileItem struct {
	FileID       string `json:"file_id"`
	OriginalName string `json:"original_name"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mime_type"`
	FilehubURL   string `json:"filehub_url"`
	CreatedAt    string `json:"created_at"`
	DownloadURL  string `json:"download_url"`
}

func NewClient(cfg Config) *Client {
	return &Client{
		Endpoint: strings.TrimRight(cfg.Endpoint, "/"),
		LocalKey: cfg.LocalKey,
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) UploadFile(path string, folderID *string, progress func(int)) (FileItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return FileItem{}, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return FileItem{}, err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return FileItem{}, err
	}
	progressReader := NewProgressReader(file, stat.Size(), progress)
	if _, err := io.Copy(part, progressReader); err != nil {
		return FileItem{}, err
	}
	if err := writer.Close(); err != nil {
		return FileItem{}, err
	}

	// 构建 URL，folder_id 作为查询参数
	url := c.Endpoint + "/api/v1/files"
	if folderID != nil {
		url += "?folder_id=" + *folderID
	}

	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return FileItem{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return FileItem{}, err
	}
	defer resp.Body.Close()
	return decodeFileResponse(resp)
}

func (c *Client) DownloadFile(fileID, outputPath string, progress func(int)) (string, error) {
	req, err := http.NewRequest("GET", c.Endpoint+"/api/v1/files/"+fileID+"/download", nil)
	if err != nil {
		return "", err
	}
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}
	if outputPath == "" {
		outputPath = "."
	}
	filename := extractFilename(resp.Header.Get("Content-Disposition"))
	if filename == "" {
		filename = fileID
	}
	if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		outputPath = filepath.Join(outputPath, filename)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", err
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	contentLength := resp.ContentLength
	if contentLength <= 0 {
		contentLength = 0
	}
	progressReader := NewProgressReader(resp.Body, contentLength, progress)
	if _, err := io.Copy(file, progressReader); err != nil {
		return "", err
	}
	return outputPath, nil
}

func (c *Client) ListFiles(folderID *string, limit, offset int, order, keyword string) ([]FileItem, int, error) {
	url := fmt.Sprintf("%s/api/v1/files?limit=%d&offset=%d&order=%s", c.Endpoint, limit, offset, order)
	if folderID != nil {
		url += "&folder_id=" + *folderID
	}
	if keyword != "" {
		url += "&keyword=" + keyword
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("list failed: %s", resp.Status)
	}
	var payload APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, 0, err
	}
	if payload.Code != 0 {
		return nil, 0, errors.New(payload.Message)
	}
	var data struct {
		Total int        `json:"total"`
		Files []FileItem `json:"files"`
	}
	if err := json.Unmarshal(payload.Data, &data); err != nil {
		return nil, 0, err
	}
	return data.Files, data.Total, nil
}

func (c *Client) GetFile(fileID string) (*FileItem, error) {
	req, err := http.NewRequest("GET", c.Endpoint+"/api/v1/files/"+fileID, nil)
	if err != nil {
		return nil, err
	}
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get file failed: %s", resp.Status)
	}
	var payload APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, errors.New(payload.Message)
	}
	var result FileItem
	if err := json.Unmarshal(payload.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteFile(fileID string) error {
	req, err := http.NewRequest("DELETE", c.Endpoint+"/api/v1/files/"+fileID, nil)
	if err != nil {
		return err
	}
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) ShareFile(fileID string) (string, error) {
	req, err := http.NewRequest("GET", c.Endpoint+"/api/v1/files/"+fileID+"/share", nil)
	if err != nil {
		return "", err
	}
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return decodeShareURL(resp)
}

func (c *Client) attachLocalKey(req *http.Request) {
	if c.LocalKey != "" {
		req.Header.Set("X-Local-Key", c.LocalKey)
	}
}

func decodeFileResponse(resp *http.Response) (FileItem, error) {
	if resp.StatusCode != http.StatusOK {
		return FileItem{}, fmt.Errorf("upload failed: %s", resp.Status)
	}
	var payload APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return FileItem{}, err
	}
	if payload.Code != 0 {
		return FileItem{}, errors.New(payload.Message)
	}
	var data FileItem
	if err := json.Unmarshal(payload.Data, &data); err != nil {
		return FileItem{}, err
	}
	return data, nil
}

func decodeShareURL(resp *http.Response) (string, error) {
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("share failed: %s", resp.Status)
	}
	var payload APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Code != 0 {
		return "", errors.New(payload.Message)
	}
	var data struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(payload.Data, &data); err != nil {
		return "", err
	}
	return data.URL, nil
}

func extractFilename(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "filename=")
	if len(parts) < 2 {
		return ""
	}
	name := strings.Trim(parts[1], "\"; ")
	return name
}

// FolderItem 文件夹信息
type FolderItem struct {
	FolderID  string  `json:"folder_id"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	ItemCount int     `json:"item_count,omitempty"`
	CreatedAt string  `json:"created_at"`
}

// CreateFolder 创建文件夹
func (c *Client) CreateFolder(name string, parentID *string) (*FolderItem, error) {
	body := map[string]interface{}{
		"name": name,
	}
	if parentID != nil {
		body["parent_id"] = *parentID
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.Endpoint+"/api/v1/folders", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create folder failed: %s", resp.Status)
	}
	var payload APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, errors.New(payload.Message)
	}
	var result FolderItem
	if err := json.Unmarshal(payload.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListFolders 列出文件夹
func (c *Client) ListFolders(parentID *string) ([]FolderItem, error) {
	url := c.Endpoint + "/api/v1/folders"
	if parentID != nil {
		url += "?parent_id=" + *parentID
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list folders failed: %s", resp.Status)
	}
	var payload APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, errors.New(payload.Message)
	}
	var data struct {
		Folders []FolderItem `json:"folders"`
	}
	if err := json.Unmarshal(payload.Data, &data); err != nil {
		return nil, err
	}
	return data.Folders, nil
}

// GetFolder 获取文件夹信息（使用 contents API）
func (c *Client) GetFolder(folderID string) (*FolderContents, error) {
	req, err := http.NewRequest("GET", c.Endpoint+"/api/v1/folders/"+folderID+"/contents", nil)
	if err != nil {
		return nil, err
	}
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get folder failed: %s", resp.Status)
	}
	var payload APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, errors.New(payload.Message)
	}
	var result FolderContents
	if err := json.Unmarshal(payload.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteFolder 删除文件夹
func (c *Client) DeleteFolder(folderID string) error {
	req, err := http.NewRequest("DELETE", c.Endpoint+"/api/v1/folders/"+folderID, nil)
	if err != nil {
		return err
	}
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete folder failed: %s", resp.Status)
	}
	return nil
}

// RenameFolder 重命名文件夹
func (c *Client) RenameFolder(folderID, newName string) error {
	body := "name=" + newName
	req, err := http.NewRequest("PUT", c.Endpoint+"/api/v1/folders/"+folderID, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rename folder failed: %s", resp.Status)
	}
	return nil
}

// MoveFolder 移动文件夹
func (c *Client) MoveFolder(folderID string, parentID *string) error {
	body := map[string]interface{}{}
	if parentID != nil {
		body["parent_id"] = *parentID
	} else {
		body["parent_id"] = nil
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", c.Endpoint+"/api/v1/folders/"+folderID+"/move", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("move folder failed: %s", resp.Status)
	}
	return nil
}

// FolderContents 文件夹内容
type FolderContents struct {
	FolderID    string           `json:"folder_id"`
	Name        string           `json:"name"`
	ParentID    *string          `json:"parent_id"`
	CreatedAt   string           `json:"created_at"`
	Folders     []FolderItem     `json:"folders"`
	Files       []FileItem       `json:"files"`
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

// GetFolderContents 获取文件夹内容
func (c *Client) GetFolderContents(folderID string) (*FolderContents, error) {
	req, err := http.NewRequest("GET", c.Endpoint+"/api/v1/folders/"+folderID+"/contents", nil)
	if err != nil {
		return nil, err
	}
	c.attachLocalKey(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get folder contents failed: %s", resp.Status)
	}
	var payload APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, errors.New(payload.Message)
	}
	var result FolderContents
	if err := json.Unmarshal(payload.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
