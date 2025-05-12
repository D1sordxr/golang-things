package internal

import (
	"context"
	appSrv "golang-things/with-worker-pool/internal/presentation/http"
	"os"
	"os/signal"
	"syscall"
)

type App struct {
	Server *appSrv.Server
}

func NewApp() *App {
	server := appSrv.NewServer()
	return &App{
		Server: server,
	}
}

func (a *App) Run() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		err := a.Server.StartServer()
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		// log.Info("Received shutdown signal, shutting down...")
		// a.Server.Shutdown()
	case _ = <-errChan:
		// log.Error("Server error: %v", err)
	}
}
