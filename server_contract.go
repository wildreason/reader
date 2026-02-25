package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// serveContractHTML starts an HTTP server for a contract file with its own renderer
func serveContractHTML(filePath string, port int) {
	absPath, _ := filepath.Abs(filePath)

	var (
		mu          sync.RWMutex
		currentHTML string
		broadcaster = newSSEBroadcaster()
	)

	// Initial render
	buildPage := func() string {
		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Sprintf("<pre>Error reading file: %s</pre>", err.Error())
		}
		fm, body := ParseFrontmatter(string(content))
		title := filepath.Base(filePath)
		if fm.Title != "" {
			title = fm.Title
		}
		return RenderContractHTMLPage(title, body, fm)
	}

	currentHTML = buildPage()

	// File watcher
	go func() {
		var lastMod time.Time
		for {
			time.Sleep(500 * time.Millisecond)
			stat, err := os.Stat(absPath)
			if err != nil {
				continue
			}
			if stat.ModTime().After(lastMod) {
				if !lastMod.IsZero() {
					rendered := buildPage()
					mu.Lock()
					currentHTML = rendered
					mu.Unlock()
					broadcaster.notify()
				}
				lastMod = stat.ModTime()
			}
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
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

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	addr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(os.Stderr, "Serving %s at http://localhost:%d\n", filePath, port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
