package internal

import (
	"context"
	appSrv "golang-things/with-worker-pool/internal/presentation/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type App struct {
	Server *appSrv.Server
}

func NewApp() *App {
	server := appSrv.NewServer("8080")
	return &App{
		Server: server,
	}
}

func (a *App) Run() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	appsWg := &sync.WaitGroup{}
	errChan := make(chan error, 1)

	appsWg.Add(1)
	go func() {
		defer appsWg.Done()
		err := a.Server.StartServer()
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		// log.Info("Received shutdown signal, shutting down...")
	case err := <-errChan:
		_ = err
		// log.Error("Server error: %v", err)
	}

	appsWg.Wait()
	// log.Info("Shutting down gracefully...")
}
