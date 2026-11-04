package server_handler

import "github.com/gin-gonic/gin"

type ServerHandler interface {
	ServerOk(ctx *gin.Context)
}

type serverHandler struct {
}

// ServerOk implements ServerHandler.
// @Summary Check server health
// @Description Public health endpoint; no API key is required.
// @Tags Server
// @Produce json
// @Success 200 {object} gin.H{status=string} "Server is running"
// @Router /server/ok [get]
func (s *serverHandler) ServerOk(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"status": "ok",
	})
}

func NewServerHandler() ServerHandler {
	return &serverHandler{}
}
