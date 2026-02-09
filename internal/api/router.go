package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kiry163/filehub/internal/service"
	"github.com/kiry163/filehub/web"
)

func NewRouter(svc *service.Service) *gin.Engine {
	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	if svc.Config.Upload.MaxSizeMB > 0 {
		router.MaxMultipartMemory = svc.Config.Upload.MaxSizeMB * 1024 * 1024
	}

	registerWebRoutes(router)

	handler := &Handler{Service: svc}
	router.GET("/health", handler.Health)

	api := router.Group("/api/v1")
	auth := api.Group("/auth")
	auth.POST("/login", handler.Login)
	auth.POST("/refresh", handler.Refresh)
	auth.POST("/logout", AuthMiddleware(svc), handler.Logout)

	api.GET("/files/:id/preview", handler.PreviewFile)
	api.GET("/files/stream", handler.StreamFile)

	files := api.Group("/files")
	files.Use(AuthMiddleware(svc))
	files.POST("", handler.UploadFile)
	files.GET("", handler.ListFiles)
	files.GET("/:id", handler.GetFile)
	files.GET("/:id/download", handler.DownloadFile)
	files.DELETE("/:id", handler.DeleteFile)
	files.GET("/:id/share", handler.ShareFile)

	return router
}

func registerWebRoutes(router *gin.Engine) {
	dist, err := web.DistFS()
	if err != nil {
		return
	}
	fileSystem := http.FS(dist)
	fileServer := http.FileServer(fileSystem)

	serveIndex := func(c *gin.Context) {
		content, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to load index.html")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	}

	router.GET("/", serveIndex)

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			c.Status(http.StatusNotFound)
			return
		}
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath != "" && cleanPath != "/" {
			if _, err := fs.Stat(dist, cleanPath); err == nil {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		serveIndex(c)
	})
}
