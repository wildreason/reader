package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// sseClient represents a connected SSE client
type sseClient struct {
	ch chan struct{}
}

// sseBroadcaster manages SSE client subscriptions
type sseBroadcaster struct {
	mu      sync.Mutex
	clients map[*sseClient]struct{}
}

func newSSEBroadcaster() *sseBroadcaster {
	return &sseBroadcaster{
		clients: make(map[*sseClient]struct{}),
	}
}

func (b *sseBroadcaster) subscribe() *sseClient {
	c := &sseClient{ch: make(chan struct{}, 1)}
	b.mu.Lock()
	b.clients[c] = struct{}{}
	b.mu.Unlock()
	return c
}

func (b *sseBroadcaster) unsubscribe(c *sseClient) {
	b.mu.Lock()
	delete(b.clients, c)
	b.mu.Unlock()
}

func (b *sseBroadcaster) notify() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for c := range b.clients {
		select {
		case c.ch <- struct{}{}:
		default:
		}
	}
}

// serveHTML starts an HTTP server serving the rendered file
func serveHTML(filePath string, blocks []Block, port int) {
	var (
		mu           sync.RWMutex
		currentHTML  string
		broadcaster  = newSSEBroadcaster()
		title        = filepath.Base(filePath)
	)

	// Initial render
	currentHTML = RenderHTMLPage(title, blocks, showLineNumbers)

	// File watcher: re-parse + re-render on change, notify SSE clients
	if filePath != "" && filePath != "stdin" {
		stopCh := make(chan struct{})
		defer close(stopCh)

		go watchAndRerender(filePath, title, &mu, &currentHTML, broadcaster, stopCh)
	}

	// GET / -- serve rendered HTML
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		mu.RLock()
		page := currentHTML
		mu.RUnlock()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, page)
	})

	// GET /events -- SSE endpoint
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher.Flush()

		client := broadcaster.subscribe()
		defer broadcaster.unsubscribe(client)

		for {
			select {
			case <-client.ch:
				fmt.Fprintf(w, "data: reload\n\n")
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(os.Stderr, "Serving %s at http://localhost:%d\n", filePath, port)

	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// watchAndRerender polls the file for changes, re-parses, re-renders HTML, and notifies SSE clients
func watchAndRerender(filePath string, title string, mu *sync.RWMutex, currentHTML *string, broadcaster *sseBroadcaster, stopCh <-chan struct{}) {
	parser := detectParser(filePath)
	var lastModTime time.Time

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		time.Sleep(500 * time.Millisecond)

		stat, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		if stat.ModTime().After(lastModTime) {
			lastModTime = stat.ModTime()

			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			blocks := parser.Parse(string(content))
			if len(blocks) == 0 {
				continue
			}

			rendered := RenderHTMLPage(title, blocks, showLineNumbers)

			mu.Lock()
			*currentHTML = rendered
			mu.Unlock()

			broadcaster.notify()
		}
	}
}
