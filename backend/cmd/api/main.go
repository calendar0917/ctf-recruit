package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ctf/backend/internal/app"
	"ctf/backend/internal/config"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	server, err := app.NewServer(cfg)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			emit("warn", "api.close.failed", map[string]any{"error": err.Error()})
		}
	}()

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go server.StartBackground(context.Background())

	go func() {
		emit("info", "api.started", map[string]any{"addr": cfg.HTTPAddr})
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	emit("info", "api.shutdown.signal", map[string]any{"signal": sig.String()})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		emit("error", "api.shutdown.failed", map[string]any{"error": err.Error()})
	}
}

func emit(level, event string, fields map[string]any) {
	entry := map[string]any{
		"level": level,
		"time":  time.Now().UTC().Format(time.RFC3339Nano),
		"event": event,
	}
	for key, value := range fields {
		entry[key] = value
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		log.Printf("structured log fallback: event=%s fields=%v", event, fields)
		return
	}
	log.Print(string(payload))
}
