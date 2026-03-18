package loki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Entry represents a single log entry to push to Loki.
type Entry struct {
	Timestamp time.Time
	Line      string
	Labels    map[string]string
}

// Client batches log entries and pushes them to the Loki HTTP API.
type Client struct {
	url        string
	httpClient *http.Client

	mu      sync.Mutex
	batch   []Entry
	maxSize int
	flushCh chan struct{}
	done    chan struct{}
}

// NewClient creates a new Loki client that batches entries and flushes
// when maxBatchSize is reached or flushInterval elapses.
func NewClient(lokiURL string, maxBatchSize int, flushInterval time.Duration) *Client {
	c := &Client{
		url:        lokiURL + "/loki/api/v1/push",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		batch:      make([]Entry, 0, maxBatchSize),
		maxSize:    maxBatchSize,
		flushCh:    make(chan struct{}, 1),
		done:       make(chan struct{}),
	}

	go c.flushLoop(flushInterval)
	return c
}

// Send adds an entry to the batch. If the batch reaches maxSize, it triggers a flush.
func (c *Client) Send(e Entry) {
	c.mu.Lock()
	c.batch = append(c.batch, e)
	shouldFlush := len(c.batch) >= c.maxSize
	c.mu.Unlock()

	if shouldFlush {
		select {
		case c.flushCh <- struct{}{}:
		default:
		}
	}
}

// Stop flushes remaining entries and stops the flush loop.
func (c *Client) Stop() {
	close(c.done)
	c.Flush()
}

// Flush sends all buffered entries to Loki.
func (c *Client) Flush() {
	c.mu.Lock()
	if len(c.batch) == 0 {
		c.mu.Unlock()
		return
	}
	entries := c.batch
	c.batch = make([]Entry, 0, c.maxSize)
	c.mu.Unlock()

	if err := c.push(entries); err != nil {
		log.Printf("loki push error: %v", err)
	} else {
		log.Printf("pushed %d entries to loki", len(entries))
	}
}

func (c *Client) flushLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.Flush()
		case <-c.flushCh:
			c.Flush()
		}
	}
}

// lokiPushRequest matches the Loki push API JSON format.
type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

func (c *Client) push(entries []Entry) error {
	// Group entries by label set
	streams := make(map[string]*lokiStream)
	for _, e := range entries {
		key := labelsKey(e.Labels)
		s, ok := streams[key]
		if !ok {
			s = &lokiStream{
				Stream: e.Labels,
				Values: make([][]string, 0),
			}
			streams[key] = s
		}
		ts := strconv.FormatInt(e.Timestamp.UnixNano(), 10)
		s.Values = append(s.Values, []string{ts, e.Line})
	}

	req := lokiPushRequest{
		Streams: make([]lokiStream, 0, len(streams)),
	}
	for _, s := range streams {
		req.Streams = append(req.Streams, *s)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request error: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("loki returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func labelsKey(labels map[string]string) string {
	// Simple key generation for grouping
	b, _ := json.Marshal(labels)
	return string(b)
}
