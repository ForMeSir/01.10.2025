package service

import (
	"context"
)

type Download interface {
	DownloadFile(url string, task *Task) (err error)
	GetTask(taskID string) (*Task, bool)
	CreateTask(urls []string) *Task
	UpdateTaskStatus(taskID string, status string)
	StartTask(ctx context.Context, taskID string)
	LoadTasks() (tasks map[string]*Task, err error)
	SaveTasks() (err error)
	Shutdown(ctx context.Context) error
}

type Service struct {
	Load Download
}

func NewService() *Service {
	return &Service{
		Load: NewLoadService(),
	}
}
