package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type downloadFileInput struct {
	Files []string `json:"files"`
}

func (h *Handler) downloadFile(c *gin.Context) {
	var input downloadFileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	task := h.services.Load.CreateTask(input.Files)

	ctx := context.Background()

	h.services.Load.StartTask(ctx, task.ID)

	c.JSON(http.StatusOK, task)
}

type taskStatusInput struct {
	Id string `json:"id"`
}

func (h *Handler) taskStatus(c *gin.Context) {
	id := c.Param("id")

	task, exists := h.services.Load.GetTask(id)
	if !exists {
		newErrorResponse(c, http.StatusNotFound, "Task not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":           task.Status,
		"failedToDowmload": task.Failed,
	})
}
