package app

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type logEntry struct {
	Level  string         `json:"level"`
	Time   string         `json:"time"`
	Event  string         `json:"event"`
	Fields map[string]any `json:"fields,omitempty"`
}

var logger = newStructuredLogger(os.Stdout)

type structuredLogger struct {
	mu  sync.Mutex
	out io.Writer
}

func newStructuredLogger(out io.Writer) *structuredLogger {
	return &structuredLogger{out: out}
}

func (l *structuredLogger) Log(level, event string, fields map[string]any) {
	if l == nil {
		return
	}
	entry := logEntry{
		Level:  level,
		Time:   time.Now().UTC().Format(time.RFC3339Nano),
		Event:  event,
		Fields: fields,
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		log.Printf("failed to encode structured log: %v", err)
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.out.Write(append(payload, '\n'))
}

func logInfo(event string, fields map[string]any) {
	logger.Log("info", event, fields)
}

func logWarn(event string, fields map[string]any) {
	logger.Log("warn", event, fields)
}

func logError(event string, fields map[string]any) {
	logger.Log("error", event, fields)
}
