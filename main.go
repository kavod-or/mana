package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mana/internal/web"
)

//go:embed content web/templates/index.html web/static/*
var assets embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	content, err := loadContentFS()
	if err != nil {
		logger.Error("could not load content", "error", err)
		os.Exit(1)
	}

	events, err := loadEventFS(content)
	if err != nil {
		logger.Error("could not load events", "error", err)
		os.Exit(1)
	}

	staticFiles, err := fs.Sub(assets, "web/static")
	if err != nil {
		logger.Error("could not load static files", "error", err)
		os.Exit(1)
	}

	handler, err := web.New(events, assets, staticFiles, logger, content)
	if err != nil {
		logger.Error("could not create web server", "error", err)
		os.Exit(1)
	}

	port := envOrDefault("PORT", "8080")
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownContext.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	}()

	logger.Info("mana is ready", "address", fmt.Sprintf("http://localhost:%s", port))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
