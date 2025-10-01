package handler

import (
	"workmate/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	services *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{services: service}
}

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()
	router.POST("/download", h.downloadFile)
	router.GET("/status/:id", h.taskStatus)
	return router
}
