// Package server implements the optional Wish SSH server mode.
// Run with: ./fathom --serve [--port 23234] [--host 0.0.0.0]
//
// Each connecting SSH client gets its own isolated BubbleTea program — no
// shared state between sessions.
package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"

	"github.com/mogglemoss/fathom/config"
	"github.com/mogglemoss/fathom/model"
)

// teaHandler creates a fresh model per SSH connection.
func teaHandler(cfg config.Config) bm.Handler {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		opts := append(bm.MakeOptions(s), tea.WithAltScreen())
		return model.New(cfg), opts
	}
}

// Start runs the Wish SSH server, blocking until SIGINT/SIGTERM.
func Start(host string, port int, cfg config.Config) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	srv, err := wish.NewServer(
		wish.WithAddress(addr),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bm.Middleware(teaHandler(cfg)),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return fmt.Errorf("could not create SSH server: %w", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("SSH server listening", "addr", addr)
	log.Info("connect with", "cmd", fmt.Sprintf("ssh <this-host> -p %d", port))

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-done:
		log.Info("shutting down SSH server")
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
