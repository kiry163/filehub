package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kiry163/filehub/internal/service"
)

func AuthMiddleware(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := svc.ParseAccessToken(tokenString)
			if err == nil {
				c.Set("user", claims.Subject)
				c.Next()
				return
			}
		}

		localKey := c.GetHeader("X-Local-Key")
		if localKey != "" && svc.Config.Auth.LocalKey != "" && localKey == svc.Config.Auth.LocalKey {
			c.Set("user", "local")
			c.Next()
			return
		}

		Error(c, http.StatusUnauthorized, 10001, "unauthorized")
		c.Abort()
	}
}
