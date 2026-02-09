package api

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(200, APIResponse{Code: 0, Data: data})
}

func Message(c *gin.Context, message string) {
	c.JSON(200, APIResponse{Code: 0, Message: message})
}

func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, APIResponse{Code: code, Message: message})
}
