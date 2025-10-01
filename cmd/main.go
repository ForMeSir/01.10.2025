package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	server "workmate"
	"workmate/internal/handler"
	"workmate/internal/service"

	"github.com/sirupsen/logrus"
)

func main() {
	services := service.NewService()
	handlers := handler.NewHandler(services)
	tasks, err := services.Load.LoadTasks()
	if err != nil && err != io.EOF {
		logrus.Info("Error loading tasks: %s", err)
	}

	for _, task := range tasks {
		if task.Status != "completed" {
			go func(task *service.Task) {
				ctx := context.Background()
				services.Load.StartTask(ctx, task.ID)
			}(task)
		}
	}

	server := new(server.Server)
	server.NewServer("8080", handlers.InitRoutes())
	go func() {
		if err := server.Run(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("error server init %s", err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM,syscall.SIGHUP)
	<-quit
	logrus.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = services.Load.SaveTasks()
	if err != nil {
		logrus.Info("Error saving tasks: %s", err)
	}

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Forced shutdown: %s", err)
	}

	if err := services.Load.Shutdown(ctx); err != nil {
		logrus.Fatalf("Service shutdown error: %v", err)
	}

	logrus.Info("The server has terminated successfully.")
}
