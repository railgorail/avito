package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"railgorail/avito/internal/config"
	"railgorail/avito/internal/lib/sl"
)

type Server struct {
	httpServer *http.Server
	log        *slog.Logger
}

func New(handler http.Handler, log *slog.Logger, cfg *config.Config) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.HTTPServer.Address,
			Handler:      handler,
			ReadTimeout:  cfg.HTTPServer.ReadTimeout,
			WriteTimeout: cfg.HTTPServer.WriteTimeout,
			IdleTimeout:  cfg.HTTPServer.IdleTimeout,
		},
		log: log,
	}
}

func (s *Server) Run(shutdownTimeout time.Duration) {
	serverErrCh := make(chan error, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s.log.Info("starting server", slog.String("addr", s.httpServer.Addr))
		err := s.httpServer.ListenAndServe()
		serverErrCh <- err
	}()

	select {
	case sig := <-sigCh:
		s.log.Info("stopping service", slog.String("signal", sig.String()))
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.log.Error("failed to stop server gracefully", sl.Err(err))
		} else {
			s.log.Info("server stopped gracefully")
		}

	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("server error", sl.Err(err))
			os.Exit(1)
		}
		s.log.Info("server exited", slog.Any("err", err))
	}
}
