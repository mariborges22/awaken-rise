package main

import (
	"log/slog"
	"os"

	"github.com/awaken-rise/backend/internal/app"
)

func main() {
	// 1. Logging Setup (JSON)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
	slog.SetDefault(slog.New(handler))

	slog.Info("Starting Awaken SaaS API...")

	// 2. Initialize and Start App
	application := app.NewApp()
	if err := application.Start(); err != nil {
		slog.Error("Application failed to start", "error", err)
		os.Exit(1)
	}
}
