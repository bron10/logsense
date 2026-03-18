package loki

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestClient_Push(t *testing.T) {
	var (
		mu       sync.Mutex
		received []lokiPushRequest
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/loki/api/v1/push" {
			t.Errorf("expected /loki/api/v1/push, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		var req lokiPushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		mu.Lock()
		received = append(received, req)
		mu.Unlock()

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, 5, 100*time.Millisecond)
	defer client.Stop()

	// Send 3 entries
	now := time.Now()
	for i := 0; i < 3; i++ {
		client.Send(Entry{
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Line:      "test log line",
			Labels:    map[string]string{"job": "test"},
		})
	}

	// Wait for time-based flush
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) == 0 {
		t.Fatal("expected at least one push request")
	}

	req := received[0]
	if len(req.Streams) != 1 {
		t.Fatalf("expected 1 stream, got %d", len(req.Streams))
	}
	if req.Streams[0].Stream["job"] != "test" {
		t.Errorf("expected job=test label, got %v", req.Streams[0].Stream)
	}
	if len(req.Streams[0].Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(req.Streams[0].Values))
	}
}

func TestClient_BatchFlush(t *testing.T) {
	var (
		mu       sync.Mutex
		pushes   int
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		pushes++
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// Set batch size to 3 with long flush interval
	client := NewClient(server.URL, 3, 10*time.Second)
	defer client.Stop()

	now := time.Now()
	for i := 0; i < 3; i++ {
		client.Send(Entry{
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Line:      "batch test",
			Labels:    map[string]string{"job": "batch-test"},
		})
	}

	// Wait for batch-triggered flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if pushes == 0 {
		t.Error("expected batch-size flush to trigger")
	}
}

func TestClient_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, 100, 50*time.Millisecond)
	defer client.Stop()

	client.Send(Entry{
		Timestamp: time.Now(),
		Line:      "error test",
		Labels:    map[string]string{"job": "error-test"},
	})

	// Wait for flush - should log error but not panic
	time.Sleep(200 * time.Millisecond)
}

func TestClient_MultipleLabels(t *testing.T) {
	var (
		mu       sync.Mutex
		received []lokiPushRequest
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req lokiPushRequest
		json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		received = append(received, req)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, 100, 50*time.Millisecond)
	defer client.Stop()

	now := time.Now()
	client.Send(Entry{Timestamp: now, Line: "info log", Labels: map[string]string{"job": "test", "level": "info"}})
	client.Send(Entry{Timestamp: now, Line: "error log", Labels: map[string]string{"job": "test", "level": "error"}})

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) == 0 {
		t.Fatal("expected push request")
	}

	// Should have 2 separate streams (different label sets)
	req := received[0]
	if len(req.Streams) != 2 {
		t.Errorf("expected 2 streams for different labels, got %d", len(req.Streams))
	}
}
