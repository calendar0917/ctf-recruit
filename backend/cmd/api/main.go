package main

import (
	"ctf-recruit/backend/internal/config"
	"ctf-recruit/backend/internal/platform"
	"ctf-recruit/backend/internal/router"
	"log/slog"
	"os"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	appCtx, err := platform.NewApp(cfg)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	router.Register(appCtx)

	if err := appCtx.App.Listen(":" + cfg.Port); err != nil {
		slog.Error("failed to start server", "error", err, "port", cfg.Port)
		os.Exit(1)
	}
}
