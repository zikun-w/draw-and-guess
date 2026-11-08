package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"drawguess/internal/game"
	"drawguess/internal/httpapi"
	"drawguess/internal/storage"
)

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	maxRooms, err := strconv.Atoi(env("MAX_ROOMS", "1000"))
	if err != nil || maxRooms < 1 || maxRooms > 1000 {
		slog.Error("MAX_ROOMS must be between 1 and 1000")
		os.Exit(1)
	}
	store, err := storage.Open(env("DATABASE_PATH", "data/game.db"))
	if err != nil {
		slog.Error("open database", "error", err)
		os.Exit(1)
	}
	hub := game.NewHubWithLimit(store, maxRooms)
	api := httpapi.New(hub, store, os.Getenv("COOKIE_SECURE") == "true", os.Getenv("TRUST_PROXY") == "true")
	srv := &http.Server{Addr: env("LISTEN_ADDR", "127.0.0.1:8080"), Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	failures := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "error", err)
			failures <- err
		}
	}()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	running := true
	var runError error
	for running {
		select {
		case <-ticker.C:
			api.Cleanup(time.Now())
		case <-ctx.Done():
			running = false
		case runError = <-failures:
			running = false
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
	hub.Close()
	if err := store.Close(); err != nil {
		slog.Error("close database", "error", err)
	}
	slog.Info("server stopped")
	if runError != nil {
		os.Exit(1)
	}
}
