package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/logsense/parser/internal/loki"
	"github.com/logsense/parser/internal/parser"
	"github.com/logsense/parser/internal/tailer"
)

func main() {
	lokiURL := envOrDefault("LOKI_URL", "http://loki:3100")
	logDir := envOrDefault("LOG_DIR", "/var/log/logsense")
	jobLabel := envOrDefault("JOB_LABEL", "logsense-parsed")
	pollMs := envOrDefault("POLL_INTERVAL_MS", "500")

	pollInterval := 500 * time.Millisecond
	if ms, err := time.ParseDuration(pollMs + "ms"); err == nil {
		pollInterval = ms
	}

	log.Printf("starting logsense parser: loki=%s dir=%s job=%s poll=%v", lokiURL, logDir, jobLabel, pollInterval)

	// Discover .log files
	files, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	if err != nil {
		log.Fatalf("failed to discover log files: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("no .log files found in %s", logDir)
	}
	log.Printf("discovered %d log files: %v", len(files), files)

	// Create Loki client (batch 100, flush every 1s)
	client := loki.NewClient(lokiURL, 100, 1*time.Second)

	// Create parser
	p := parser.New()

	// Graceful shutdown
	done := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start a goroutine per file
	for _, f := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			processFile(path, p, client, jobLabel, pollInterval, done)
		}(f)
	}

	// Wait for shutdown signal
	sig := <-sigs
	log.Printf("received signal %v, shutting down", sig)
	close(done)
	wg.Wait()
	client.Stop()
	log.Println("shutdown complete")
}

func processFile(path string, p *parser.Parser, client *loki.Client, jobLabel string, pollInterval time.Duration, done <-chan struct{}) {
	filename := filepath.Base(path)
	source := strings.TrimSuffix(filename, filepath.Ext(filename))
	log.Printf("tailing %s (source=%s)", path, source)

	lines := tailer.Tail(path, pollInterval, done)

	for line := range lines {
		result := p.Parse(line)
        fields := result.Fields

        // metadata
        fields["parse_strategy"] = result.Strategy
        fields["parse_status"] = result.Status
		// Build structured JSON line
		structured, err := json.Marshal(fields)
		if err != nil {
			log.Printf("marshal error for line in %s: %v", filename, err)
			continue
		}

		// Extract level for label, default to "unknown"
		level := "unknown"
		if l, ok := fields["level"]; ok {
			level = strings.ToLower(l)
		}

		client.Send(loki.Entry{
			Timestamp: time.Now(),
			Line:      string(structured),
			Labels: map[string]string{
				"job":    jobLabel,
				"source": source,
				"level":  level,
			},
		})
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
