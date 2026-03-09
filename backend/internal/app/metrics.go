package app

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type metricsRegistry struct {
	mu       sync.Mutex
	counters map[string]float64
}

func newMetricsRegistry() *metricsRegistry {
	return &metricsRegistry{counters: make(map[string]float64)}
}

func (m *metricsRegistry) Inc(name string, labels map[string]string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[metricKey(name, labels)]++
}

func (m *metricsRegistry) Add(name string, value float64, labels map[string]string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[metricKey(name, labels)] += value
}

func (m *metricsRegistry) snapshot() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	lines := make([]string, 0, len(m.counters))
	for key, value := range m.counters {
		lines = append(lines, fmt.Sprintf("%s %g", key, value))
	}
	sort.Strings(lines)
	return lines
}

func (m *metricsRegistry) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	for _, line := range m.snapshot() {
		_, _ = fmt.Fprintln(w, line)
	}
}

func metricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", key, labels[key]))
	}
	return fmt.Sprintf("%s{%s}", name, strings.Join(parts, ","))
}
