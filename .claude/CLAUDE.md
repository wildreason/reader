# CLAUDE.md

## Project Overview

**aster** - render any file as a clean web page, instantly, from the terminal. Single binary. Any content format → readable, shareable web. Started as a terminal viewer; the web renderer is the actual product.

```bash
aster file.md --port 3000  # Web: serve rendered HTML on localhost
aster data.csv --port 3000 # Web: sortable/filterable table with auto-chart
aster demo.webm --port 3000 # Web: branded video player with controls
aster file.md --html       # Static: self-contained HTML to stdout
aster demo.mp4 --html      # Static: self-contained HTML video page
aster ~/dropstore/docs/ --port 8080  # Web: directory index with per-doc routes
aster file.md              # Terminal: Markdown with colors and tables
aster photo.png            # Terminal: Image inline (chafa)
aster changes.diff         # Terminal: Diff with syntax highlighting
aster data.csv             # Terminal: CSV as formatted table
aster data.jsonl           # Terminal: JSONL transcript viewer
aster demo.mp4             # Terminal: Video metadata (ffprobe)
aster pick                 # Pick from recent files
aster latest               # Open newest file in cwd
aster -n file.md           # Show source file line numbers
aster ~/dropstore/docs/              # Terminal: list docs with title/date/tags
```

Shell alias: `alias as=aster`

## Build

```bash
make build     # Build with version injection
make test      # Run tests
make install   # Install to ~/.local/bin
```

## Flags

- `-n` — Source file line numbers in gutter (dim gray, right-aligned)
- `-f` — Follow mode (file watching)
- `-t TYPE` — Force content type (md, json, jsonl, diff, txt, yaml, csv)
- `--port N` — Serve rendered file as HTML on localhost:N with live reload
- `--html` — Export self-contained HTML to stdout (no CDN, no server)

## Architecture

One unified pipeline: every content type follows `Parse -> []Block -> Render -> HTML`.

```
main.go              Args, subcommands, routing, flag parsing
     |
detectFileType()     Route by extension: img/vid -> FileParser, text -> Parser
     |
detectParser()       Auto-detect: .md .jsonl .diff .txt .json .csv + contract (frontmatter)
     |
parser.Parse()       Extract blocks from content (all types produce []Block)
     |
     +-- TUI path:   reader.go -> formatter.go (tview tags, Catppuccin dark)
     |
     +-- Web path:   server.go -> serveHTML() -> formatBlockHTML() (HTML/CSS/JS, brand light theme)
     |
     +-- Static:     formatter_html.go -> RenderStaticHTMLPage (inlined CSS/JS, no CDN)
```

### Block pipeline

All content types produce `[]Block` with typed payloads via `Block.Data`:

| Content type | Parser | Block.ContentType | Block.Data |
|-------------|--------|-------------------|------------|
| Markdown | MarkdownParser | BlockContentPlain | - |
| Diffs | DiffParser | BlockContentDiff | - |
| CSV/TSV | CsvParser | BlockContentCSV | *CsvData |
| JSONL | JSONLParser | BlockContentTranscript | *TranscriptData |
| Images | ImageParser (FileParser) | BlockContentImage | *ImageData |
| Video | VideoParser (FileParser) | BlockContentVideo | *VideoData |
| Contracts | ContractParser | BlockContentContract | *ContractData |
| JSON | TxtParser | BlockContentJSON | - |
| YAML | TxtParser | BlockContentYAML | - |

`formatBlockHTML()` dispatches all content types. `serveHTML()` is the only server function (plus `serveDirectory` for collections). Binary content (images, video) uses `/asset/{filename}` routes registered by `serveHTML`.

### Web mode (`--port`)

- Single block rendering: markdown files render as one continuous document (headings stay as native h1/h2/h3)
- SSE live reload: file watcher polls every 500ms, pushes reload event to all connected browsers
- No external dependencies at runtime (highlight.js + fonts loaded from CDN)
- Frontmatter: `---` delimited YAML stripped from content, `title` used in `<title>` tag

### Directory mode (`aster <dir> --port`)

- Scans `*.md` files, parses frontmatter from each
- Index page at `/` lists all docs sorted by created date desc
- Individual docs served at `/{slug}` (slug = filename without .md)
- `docCache` holds pre-rendered HTML per slug, updated by directory watcher
- SSE live reload: detects new/modified/deleted files, refreshes index + docs
- Terminal fallback: `aster <dir>` prints formatted table to stdout

### Web features

| Feature | Implementation |
|---------|---------------|
| Syntax highlighting | highlight.js CDN, `github` light theme, auto-language detection |
| Copy button | Appears on code block hover, copies to clipboard |
| Sortable tables | Click header to sort asc/desc, numeric-aware |
| Links | Open in new tab, external icon, URL tooltip on hover |
| Images | `![alt](url)` renders as `<img>`, click to expand |
| TOC sidebar | Fixed left nav from h1/h2/h3, scroll-spy, collapsible |
| Diffs | Side-by-side two-column, collapsible hunks, word-level LCS highlighting |
| Search | `/` or `Ctrl+K` opens fuzzy search overlay with arrow key navigation |
| CSV tables | Per-column filter inputs, row count, numeric right-alignment |
| Auto-chart | SVG line chart when CSV has label + numeric columns (brand colors) |
| Video player | `<video controls>`, speed buttons (0.5x/1x/1.5x/2x), keyboard shortcuts (Space/F/arrows) |

### Brand theme

- Fonts: Inter 400/600 (body), JetBrains Mono 400/600 (code)
- Colors: Navy #0A1628, Slate #1E293B, Accent Blue #3B82F6, Surface #F8FAFC, White #FFFFFF, Border #E2E8F0
- Type scale: H1 30px, H2 24px, H3 20px, Body 16px, Small 14px, Caption 12px
- Rules: Semibold for headings (no italic), Accent Blue for interactive elements only

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry, subcommand routing, auto-detect, flag parsing |
| `data_types.go` | Typed Block payloads: ImageData, VideoData, CsvData, TranscriptData, ContractData |
| `viewer_img.go` | TUI image rendering (chafa/imgcat, iterm/kitty/symbols) |
| `viewer_video.go` | TUI video rendering (ffprobe metadata, ffplay playback) |
| `parser.go` | Parser/FileParser interfaces, Block struct, BlockIndex |
| `parser_image.go` | ImageParser (FileParser for images, base64/asset modes) |
| `parser_video.go` | VideoParser (FileParser for video, base64/asset modes) |
| `parser_contract.go` | ContractParser (contracts detected via frontmatter type) |
| `parser_jsonl.go` | JSONLParser (transcripts) |
| `parser_diff.go` | DiffParser (unified diffs) |
| `parser_txt.go` | TxtParser (plain text) |
| `parser_csv.go` | CsvParser (CSV/TSV with auto-delimiter detection) |
| `parser_todo.go` | TodoParser (JSON todos) |
| `reader.go` | Scrollable TUI viewer |
| `follower.go` | Follow mode (-f), file watching |
| `formatter.go` | TUI block rendering, markdown, tables, line number gutter |
| `formatter_diff.go` | TUI diff coloring (ANSI) |
| `formatter_html.go` | Web rendering: HTML/CSS/JS, brand theme, all web features, index page |
| `formatter_contract.go` | Contract clause parsing and HTML helpers |
| `formatter_shell.go` | Shell output styling |
| `frontmatter.go` | YAML frontmatter parser (title, created, tags) |
| `frontmatter_test.go` | Frontmatter parsing tests |
| `server.go` | HTTP server (serveHTML + serveDirectory), SSE broadcaster, file watcher |
| `content_type.go` | Block content type constants and detection |
| `commands.go` | Navigator, command parsing |
| `recent.go` | Recent file history (pick/latest) |
| `context_git.go` | Git context for diffs |
| `keybindings.go` | Key action parsing |
| `embed.go` | go:embed declarations for highlight.js assets |
| `embed/` | highlight.min.js + github.min.css for static HTML export |

## Commands

```
aster <file>        View file (auto-detect format)
aster <dir>         List directory docs (table to stdout)
aster <dir> --port  Serve directory as web index with doc routes
aster pick | p      Pick from recent files
aster latest | l    Open newest file in cwd
aster help          Show help
```

Type subcommands (hidden, scoped shortcuts):
```
aster md <file|-|+>     Markdown
aster img <file|-|+>    Images
aster txt <file|-|+>    Plain text
aster diff <file|-|+>   Diffs
aster json <file|-|+>   JSON
aster csv <file|-|+>    CSV/TSV
aster jsonl <file|-|+>  Transcripts
aster vid <file|-|+>    Video
```

## Navigation (TUI)

- `j/k` - scroll down/up (3 lines)
- `d/u` - half page down/up
- `g/G` - top/bottom
- `PgDn/PgUp` - full page
- `q` - quit

## Navigation (Web)

- `/` or `Ctrl+K` - search
- `Esc` - close search
- Arrow keys - navigate search results
- TOC sidebar - click to jump to section

## Parser Interface

```go
type Parser interface {
    Parse(content string) []Block
    Detect(filePath string) bool
}

// For binary content (images, video)
type FileParser interface {
    Parser
    ParseFile(filePath string, static bool) ([]Block, error)
}
```

Add new text parser: create `parser_xxx.go`, implement `Parser`, add to `detectParser()` in main.go.
Add new binary parser: create `parser_xxx.go`, implement `FileParser`, add dispatch in `viewFile()`.

Typed payloads go in `Block.Data` (defined in `data_types.go`). The `formatBlockHTML()` switch in `formatter_html.go` dispatches rendering by `Block.ContentType`.

## Release

goreleaser builds cross-platform binaries on tag push.

```bash
git tag v1.0.1
git push origin v1.0.1
# GitHub Actions runs goreleaser -> GitHub Release + Homebrew tap
```

## Frontmatter

Files with YAML frontmatter between `---` delimiters are parsed automatically:

```yaml
---
title: Document Title
created: 2026-02-23
tags: [feature, docs]
---
```

- `ParseFrontmatter(content) -> (Frontmatter, body)` in `frontmatter.go`
- Web mode: title used in `<title>` tag, frontmatter stripped from rendered content
- TUI mode: frontmatter shows as-is (not stripped)
- Directory mode: title/created/tags populate the index listing

## v2: wedoc

v1 was the terminal file viewer. v2 is the core product: render any content as a clean web page, instantly, from the terminal. Same binary, no new project.

```bash
aster file.md --port 3000          # Serve rendered HTML (v1, done)
aster file.md --html > page.html   # Static export, self-contained
aster file.md --share              # Public URL via tunnel, expires on exit
aster file.md --deploy             # Push to hosting, permanent URL
```

### Phases

| Phase | Feature | Status |
|-------|---------|--------|
| 1 | Content parity | Done. JSON/YAML syntax highlighting, images, CSV/TSV tables with auto-chart. |
| 2 | Static export (`--html`) | Done. Self-contained HTML with inlined highlight.js CSS/JS. Works offline. |
| 3 | Public sharing (`--share`) | Planned. Public URL via tunnel (cloudflare/bore). Auto-expires on exit. |
| 4 | Deploy (`--deploy`) | Planned. Push static HTML to hosting (Vercel/CF Pages). Permanent URL. |

### Web rendering status

All content types go through the unified pipeline: `Parser -> []Block -> formatBlockHTML() -> HTML`.

| Content | Web | Gap |
|---------|-----|-----|
| Markdown | Full (TOC, search, tables, syntax highlighting, diffs) | None |
| Diffs | Full (side-by-side, word-level, collapsible) | None |
| CSV/TSV | Full (sortable, filterable, auto-chart) | None |
| JSONL | Full (transcript layout, turn-based, diff/tool rendering) | None |
| Plain text | Functional | None |
| JSON | Syntax highlighted via highlight.js | None |
| YAML | Syntax highlighted via highlight.js | None |
| Images | Web `<img>` via asset route, base64 data URI in --html | None |
| Video | Web `<video>` player with speed/keyboard controls, base64 in --html (<10MB) | None |
| Contracts | Clause-structured layout with TOC (detected via frontmatter type) | None |

### What carries forward

Everything from v1: TUI viewer, `formatter_html.go` renderer, SSE + file watcher, directory mode, frontmatter parser, stdin piping.

### What dies

SuperDoc integration, .docx editing, CRUD API, collaboration, dashboard shell, `asdoc` clipboard tool.

## Testing

```bash
make test
go test -v ./...
```
