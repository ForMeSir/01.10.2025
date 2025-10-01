package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	STATUS_IN_QUEUE  = "in_queue"
	STATUS_STARTED   = "started"
	STATUS_COMPLETED = "completed"
)

type Task struct {
	ID           string   `json:"id"`
	URLs         []string `json:"urls"`
	Status       string   `json:"status"`
	Failed       []string `json:"failed"`
	LastUrlIndex int      `json:"last_url"`
}

type LoadService struct {
	Tasks      map[string]*Task `json:"tasks"`
	tasksMutex sync.Mutex
	taskID     string
	wg         sync.WaitGroup
}

func NewLoadService() *LoadService {
	return &LoadService{
		Tasks: make(map[string]*Task),
	}
}

func (l *LoadService) DownloadFile(url string, task *Task) (err error) {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	filepath := "task_" + task.ID + "_" + path.Base(resp.Request.URL.String())
	out, err := os.Create("files/" + filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (l *LoadService) CreateTask(urls []string) *Task {
	l.tasksMutex.Lock()
	defer l.tasksMutex.Unlock()

	l.taskID = fmt.Sprint(uuid.New())

	task := &Task{
		ID:     l.taskID,
		URLs:   urls,
		Status: STATUS_IN_QUEUE,
	}
	l.Tasks[l.taskID] = task
	return task
}

func (l *LoadService) UpdateTaskStatus(taskID string, status string) {
	l.tasksMutex.Lock()
	defer l.tasksMutex.Unlock()

	if task, exists := l.Tasks[taskID]; exists {
		task.Status = status
	}
}

func (l *LoadService) GetTask(taskID string) (*Task, bool) {
	l.tasksMutex.Lock()
	defer l.tasksMutex.Unlock()

	task, exists := l.Tasks[taskID]
	return task, exists
}

func (l *LoadService) StartTask(ctx context.Context, taskID string) {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()

		select {
		case <-ctx.Done():
			return
		default:
			l.UpdateTaskStatus(taskID, STATUS_STARTED)

			task, exists := l.GetTask(taskID)
			if !exists {
				return
			}

			// Скачиваем файлы
			for i := task.LastUrlIndex; i < len(task.URLs); i++ {
				select {
				case <-ctx.Done():
					logrus.Info("Task cancelled during download: %s", taskID)
					return
				default:
					err := l.DownloadFile(task.URLs[i], task)
					task.LastUrlIndex = i
					if err != nil {
						task.Failed = append(task.Failed, task.URLs[i])
						logrus.Info("File download error: %s", err)
					}
				}
			}
			l.UpdateTaskStatus(taskID, STATUS_COMPLETED)
		}
	}()
}
func (l *LoadService) SaveTasks() (err error) {
	l.tasksMutex.Lock()
	defer l.tasksMutex.Unlock()

	file, err := os.Create("tasks.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(l.Tasks)
}

func (l *LoadService) LoadTasks() (tasks map[string]*Task, err error) {
	l.tasksMutex.Lock()
	defer l.tasksMutex.Unlock()

	file, err := os.Open("tasks.json")
	if err != nil {
		if os.IsNotExist(err) {
			return tasks, nil
		}
		return tasks, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&l.Tasks)
	if err != nil {
		return tasks, err
	}

	return l.Tasks, nil
}

func (l *LoadService) Shutdown(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		l.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		logrus.Info("Shutdown timeout exceeded")
		return ctx.Err()
	case <-done:
		logrus.Info("Shutdown completed gracefully")
		return nil
	}
}
