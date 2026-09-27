package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func New(address string, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              address,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		logger: logger,
	}
}

func (server *Server) Run(ctx context.Context) error {
	errorChannel := make(chan error, 1)

	go func() {
		server.logger.Info("http server started", "address", server.httpServer.Addr)
		errorChannel <- server.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errorChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("serve http: %w", err)
	case <-ctx.Done():
		server.logger.Info("http server stopping")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	if err := <-errorChannel; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("stop http server: %w", err)
	}

	server.logger.Info("http server stopped")

	return nil
}
