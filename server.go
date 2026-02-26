package main

import (
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// formatServerAddr returns the listen address for localhost binding
func formatServerAddr(port int) string {
	return fmt.Sprintf("127.0.0.1:%d", port)
}

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

// imageMIME returns the MIME type for an image extension
func imageMIME(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

// videoMIME returns the MIME type for a video extension
func videoMIME(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".mkv":
		return "video/x-matroska"
	default:
		return "application/octet-stream"
	}
}

// serveVideoHTML starts an HTTP server serving a video file as a branded web player
func serveVideoHTML(filePath string, port int) {
	broadcaster := newSSEBroadcaster()
	absPath, _ := filepath.Abs(filePath)
	title := filepath.Base(filePath)
	ext := filepath.Ext(filePath)
	mime := videoMIME(ext)

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
					broadcaster.notify()
				}
				lastMod = stat.ModTime()
			}
		}
	}()

	mux := http.NewServeMux()

	// GET /video -- serve raw video file
	mux.HandleFunc("/video", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, absPath)
	})

	// GET /events -- SSE
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

	// GET / -- serve HTML page with video player
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, videoPlayerHTML(title, "/video", mime))
	})

	addr := formatServerAddr(port)
	fmt.Fprintf(os.Stderr, "Serving %s at http://localhost:%d\n", filePath, port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// videoPlayerHTML returns a branded HTML page with a video player
func videoPlayerHTML(title, src, mime string) string {
	title = html.EscapeString(title)
	src = html.EscapeString(src)
	mime = html.EscapeString(mime)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter:wght@400;600&family=JetBrains+Mono:wght@400;600&display=swap">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  background: #FFFFFF;
  color: #0A1628;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-height: 100vh;
  padding: 2rem;
}
.title {
  font-size: 14px;
  font-weight: 600;
  color: #64748B;
  margin-bottom: 1.5rem;
}
.video-container {
  max-width: 90vw;
  width: 100%%;
  max-width: 960px;
}
.video-container video {
  width: 100%%;
  border-radius: 6px;
  border: 1px solid #E2E8F0;
  background: #0A1628;
  outline: none;
}
.controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.75rem;
  font-size: 13px;
  color: #64748B;
  font-family: 'JetBrains Mono', monospace;
}
.speed-btn {
  background: #F8FAFC;
  border: 1px solid #E2E8F0;
  border-radius: 4px;
  padding: 0.2rem 0.5rem;
  font-size: 12px;
  font-family: 'JetBrains Mono', monospace;
  color: #64748B;
  cursor: pointer;
  transition: all 0.15s;
}
.speed-btn:hover { border-color: #3B82F6; color: #3B82F6; }
.speed-btn.active { background: #3B82F6; color: #FFFFFF; border-color: #3B82F6; }
.shortcuts {
  margin-top: 1rem;
  font-size: 12px;
  color: #94A3B8;
}
.shortcuts kbd {
  background: #F8FAFC;
  border: 1px solid #E2E8F0;
  border-radius: 3px;
  padding: 0.1rem 0.35rem;
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
}
</style>
</head>
<body>
<div class="title">%s</div>
<div class="video-container">
  <video id="player" controls>
    <source src="%s" type="%s">
  </video>
  <div class="controls">
    <span id="time-display">0:00 / 0:00</span>
    <span style="flex:1"></span>
    <button class="speed-btn" data-speed="0.5">0.5x</button>
    <button class="speed-btn active" data-speed="1">1x</button>
    <button class="speed-btn" data-speed="1.5">1.5x</button>
    <button class="speed-btn" data-speed="2">2x</button>
  </div>
  <div class="shortcuts">
    <kbd>Space</kbd> play/pause
    <kbd>F</kbd> fullscreen
    <kbd>&#x2190;</kbd> -5s
    <kbd>&#x2192;</kbd> +5s
  </div>
</div>
<script>
var v = document.getElementById('player');
var btns = document.querySelectorAll('.speed-btn');
var timeEl = document.getElementById('time-display');

btns.forEach(function(btn) {
  btn.addEventListener('click', function() {
    v.playbackRate = parseFloat(btn.dataset.speed);
    btns.forEach(function(b) { b.classList.remove('active'); });
    btn.classList.add('active');
  });
});

function fmt(s) {
  var m = Math.floor(s / 60);
  var sec = Math.floor(s %% 60);
  return m + ':' + (sec < 10 ? '0' : '') + sec;
}

v.addEventListener('timeupdate', function() {
  timeEl.textContent = fmt(v.currentTime) + ' / ' + fmt(v.duration || 0);
});

document.addEventListener('keydown', function(e) {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
  switch(e.key) {
    case ' ':
      e.preventDefault();
      v.paused ? v.play() : v.pause();
      break;
    case 'f':
      if (v.requestFullscreen) v.requestFullscreen();
      else if (v.webkitRequestFullscreen) v.webkitRequestFullscreen();
      break;
    case 'ArrowLeft':
      e.preventDefault();
      v.currentTime = Math.max(0, v.currentTime - 5);
      break;
    case 'ArrowRight':
      e.preventDefault();
      v.currentTime = Math.min(v.duration, v.currentTime + 5);
      break;
  }
});

var es = new EventSource('/events');
es.onmessage = function(e) { if (e.data === 'reload') { v.src = '/video?' + Date.now(); } };
es.onerror = function() { setTimeout(function() { location.reload(); }, 2000); };
</script>
</body>
</html>`, title, title, src, mime)
}

// serveImageHTML starts an HTTP server serving an image file as a web page
func serveImageHTML(filePath string, port int) {
	broadcaster := newSSEBroadcaster()
	absPath, _ := filepath.Abs(filePath)
	title := filepath.Base(filePath)
	ext := filepath.Ext(filePath)
	mime := imageMIME(ext)

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
					broadcaster.notify()
				}
				lastMod = stat.ModTime()
			}
		}
	}()

	mux := http.NewServeMux()

	// GET /image -- serve raw image file
	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, absPath)
	})

	// GET /events -- SSE
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

	// GET / -- serve HTML page with image
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter:wght@400;600&display=swap">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  background: #FFFFFF;
  color: #0A1628;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-height: 100vh;
  padding: 2rem;
}
.title {
  font-size: 14px;
  font-weight: 600;
  color: #64748B;
  margin-bottom: 1.5rem;
}
.img-container {
  max-width: 90vw;
}
.img-container img {
  max-width: 100%%;
  max-height: 85vh;
  border-radius: 6px;
  border: 1px solid #E2E8F0;
  cursor: pointer;
  transition: max-width 0.2s, max-height 0.2s;
}
.img-container img.expanded {
  max-width: none;
  max-height: none;
}
</style>
</head>
<body>
<div class="title">%s</div>
<div class="img-container">
  <img src="/image" alt="%s" onclick="this.classList.toggle('expanded')">
</div>
<script>
var es = new EventSource('/events');
es.onmessage = function(e) { if (e.data === 'reload') { document.querySelector('img').src = '/image?' + Date.now(); } };
es.onerror = function() { setTimeout(function() { location.reload(); }, 2000); };
</script>
</body>
</html>`, html.EscapeString(title), html.EscapeString(title), html.EscapeString(title))
	})

	addr := formatServerAddr(port)
	fmt.Fprintf(os.Stderr, "Serving %s at http://localhost:%d\n", filePath, port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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

	// Extract frontmatter title if available (for single-block markdown)
	if len(blocks) == 1 {
		fm, body := ParseFrontmatter(blocks[0].Content)
		if fm.Title != "" {
			title = fm.Title
		}
		blocks[0].Content = body
		blocks[0].Pages = []string{body}
	}

	// Initial render
	currentHTML = RenderHTMLPage(title, blocks, showLineNumbers)

	// File watcher: re-parse + re-render on change, notify SSE clients
	if filePath != "" && filePath != "stdin" {
		stopCh := make(chan struct{})
		defer close(stopCh)

		singleBlock := len(blocks) == 1
		ct := BlockContentPlain
		if singleBlock {
			ct = blocks[0].ContentType
		}
		go watchAndRerender(filePath, title, singleBlock, ct, &mu, &currentHTML, broadcaster, stopCh)
	}

	mux := http.NewServeMux()

	// GET / -- serve rendered HTML
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

	// GET /events -- SSE endpoint
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

	addr := formatServerAddr(port)
	fmt.Fprintf(os.Stderr, "Serving %s at http://localhost:%d\n", filePath, port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// docEntry holds a cached document for directory mode
type docEntry struct {
	slug     string
	fm       Frontmatter
	html     string
	modTime  time.Time
}

// serveDirectory starts an HTTP server listing all markdown files in dirPath
func serveDirectory(dirPath string, port int) {
	var (
		mu          sync.RWMutex
		cache       = make(map[string]*docEntry) // slug -> entry
		indexHTML   string
		broadcaster = newSSEBroadcaster()
		dirName     = filepath.Base(dirPath)
	)

	// Initial scan
	scanDirectory(dirPath, cache)
	indexHTML = renderIndex(dirName, cache)

	mux := http.NewServeMux()

	// GET /events -- SSE endpoint
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

	// GET / and GET /{slug}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		path := strings.TrimPrefix(r.URL.Path, "/")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if path == "" {
			fmt.Fprint(w, indexHTML)
			return
		}

		entry, ok := cache[path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, entry.html)
	})

	// Directory watcher
	go watchDirectory(dirPath, dirName, &mu, cache, &indexHTML, broadcaster)

	addr := formatServerAddr(port)
	fmt.Fprintf(os.Stderr, "Serving %s at http://localhost:%d\n", dirPath, port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// slugFromPath converts a filename to a URL slug
func slugFromPath(filePath string) string {
	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	return name
}

// scanDirectory reads all .md files and populates the cache
func scanDirectory(dirPath string, cache map[string]*docEntry) {
	files, _ := filepath.Glob(filepath.Join(dirPath, "*.md"))
	for _, f := range files {
		loadDocEntry(f, cache)
	}
}

// loadDocEntry reads a single file and adds/updates it in the cache
func loadDocEntry(filePath string, cache map[string]*docEntry) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	stat, err := os.Stat(filePath)
	if err != nil {
		return
	}

	slug := slugFromPath(filePath)
	fm, body := ParseFrontmatter(string(content))

	title := slug
	if fm.Title != "" {
		title = fm.Title
	}

	blocks := []Block{{
		Name:        title,
		Content:     body,
		Pages:       []string{body},
		TotalPages:  1,
		ContentType: BlockContentPlain,
	}}
	rendered := RenderHTMLPage(title, blocks, false)

	cache[slug] = &docEntry{
		slug:    slug,
		fm:      fm,
		html:    rendered,
		modTime: stat.ModTime(),
	}
}

// renderIndex builds the index HTML from current cache
func renderIndex(dirName string, cache map[string]*docEntry) string {
	docs := make([]DocMeta, 0, len(cache))
	for _, entry := range cache {
		title := entry.slug
		if entry.fm.Title != "" {
			title = entry.fm.Title
		}
		docs = append(docs, DocMeta{
			Slug:    entry.slug,
			Title:   title,
			Created: entry.fm.Created,
			Tags:    entry.fm.Tags,
			ModTime: entry.modTime,
		})
	}
	// Sort by created date desc, then by title
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].Created != docs[j].Created {
			return docs[i].Created > docs[j].Created
		}
		return docs[i].Title < docs[j].Title
	})
	return RenderIndexPage(dirName, docs)
}

// watchDirectory polls for file changes in dirPath and updates cache
func watchDirectory(dirPath string, dirName string, mu *sync.RWMutex, cache map[string]*docEntry, indexHTML *string, broadcaster *sseBroadcaster) {
	for {
		time.Sleep(500 * time.Millisecond)

		files, _ := filepath.Glob(filepath.Join(dirPath, "*.md"))
		currentSlugs := make(map[string]string) // slug -> filepath
		for _, f := range files {
			currentSlugs[slugFromPath(f)] = f
		}

		changed := false

		mu.RLock()
		// Check for new or modified files
		for slug, fpath := range currentSlugs {
			stat, err := os.Stat(fpath)
			if err != nil {
				continue
			}
			entry, exists := cache[slug]
			if !exists || stat.ModTime().After(entry.modTime) {
				changed = true
				break
			}
		}
		// Check for deleted files
		for slug := range cache {
			if _, exists := currentSlugs[slug]; !exists {
				changed = true
				break
			}
		}
		mu.RUnlock()

		if !changed {
			continue
		}

		mu.Lock()
		// Remove deleted entries
		for slug := range cache {
			if _, exists := currentSlugs[slug]; !exists {
				delete(cache, slug)
			}
		}
		// Add/update entries
		for slug, fpath := range currentSlugs {
			stat, err := os.Stat(fpath)
			if err != nil {
				continue
			}
			entry, exists := cache[slug]
			if !exists || stat.ModTime().After(entry.modTime) {
				loadDocEntry(fpath, cache)
			}
		}
		*indexHTML = renderIndex(dirName, cache)
		mu.Unlock()

		broadcaster.notify()
	}
}

// watchAndRerender polls the file for changes, re-parses, re-renders HTML, and notifies SSE clients
func watchAndRerender(filePath string, title string, singleBlock bool, contentType BlockContentType, mu *sync.RWMutex, currentHTML *string, broadcaster *sseBroadcaster, stopCh <-chan struct{}) {
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

			var blocks []Block
			renderTitle := title
			var rendered string
			if singleBlock {
				bodyStr := string(content)
				fm, body := ParseFrontmatter(bodyStr)
				if fm.Title != "" {
					renderTitle = fm.Title
				}
				bodyStr = body

				blocks = []Block{{
					Name:        renderTitle,
					Content:     bodyStr,
					Pages:       []string{bodyStr},
					TotalPages:  1,
					ContentType: contentType,
				}}
				rendered = RenderHTMLPage(renderTitle, blocks, showLineNumbers)
			} else {
				blocks = parser.Parse(string(content))
				if len(blocks) == 0 {
					continue
				}
				rendered = RenderHTMLPage(renderTitle, blocks, showLineNumbers)
			}

			mu.Lock()
			*currentHTML = rendered
			mu.Unlock()

			broadcaster.notify()
		}
	}
}
