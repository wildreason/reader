# CLAUDE.md

## Project Overview

**aster** - read any file in the terminal, rendered. Single binary. Two output targets: terminal TUI and web browser.

```bash
aster file.md              # Terminal: Markdown with colors and tables
aster photo.png            # Terminal: Image inline (chafa)
aster changes.diff         # Terminal: Diff with syntax highlighting
aster data.jsonl           # Terminal: JSONL transcript viewer
aster pick                 # Pick from recent files
aster latest               # Open newest file in cwd
aster -n file.md           # Show source file line numbers
aster file.md --port 3000  # Web: serve rendered HTML on localhost
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
- `--port N` — Serve rendered file as HTML on localhost:N with live reload

## Architecture

Two rendering pipelines from the same parse layer:

```
main.go              Args, subcommands, routing, flag parsing
     |
detectFileType()     Route by extension: img -> viewImage, text -> parser
     |
detectParser()       Auto-detect: .md .jsonl .diff .txt .json
     |
parser.Parse()       Extract blocks from content
     |
     +-- TUI path:   reader.go -> formatter.go (tview tags, Catppuccin dark)
     |
     +-- Web path:   server.go -> formatter_html.go (HTML/CSS/JS, brand light theme)
```

### Web mode (`--port`)

- Single block rendering: markdown files render as one continuous document (headings stay as native h1/h2/h3)
- SSE live reload: file watcher polls every 500ms, pushes reload event to all connected browsers
- No external dependencies at runtime (highlight.js + fonts loaded from CDN)

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

### Brand theme

- Fonts: Inter 400/600 (body), JetBrains Mono 400/600 (code)
- Colors: Navy #0A1628, Slate #1E293B, Accent Blue #3B82F6, Surface #F8FAFC, White #FFFFFF, Border #E2E8F0
- Type scale: H1 30px, H2 24px, H3 20px, Body 16px, Small 14px, Caption 12px
- Rules: Semibold for headings (no italic), Accent Blue for interactive elements only

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry, subcommand routing, auto-detect, flag parsing |
| `viewer_img.go` | Image rendering (chafa/imgcat, iterm/kitty/symbols) |
| `parser.go` | MarkdownParser, Block struct, BlockIndex |
| `parser_jsonl.go` | JSONLParser (transcripts) |
| `parser_diff.go` | DiffParser (unified diffs) |
| `parser_txt.go` | TxtParser (plain text) |
| `parser_todo.go` | TodoParser (JSON todos) |
| `reader.go` | Scrollable TUI viewer |
| `follower.go` | Follow mode (-f), file watching |
| `formatter.go` | TUI block rendering, markdown, tables, line number gutter |
| `formatter_diff.go` | TUI diff coloring (ANSI) |
| `formatter_html.go` | Web rendering: HTML/CSS/JS, brand theme, all web features |
| `formatter_shell.go` | Shell output styling |
| `server.go` | HTTP server, SSE broadcaster, file watcher, live reload |
| `content_type.go` | Content type detection |
| `commands.go` | Navigator, command parsing |
| `recent.go` | Recent file history (pick/latest) |
| `context_git.go` | Git context for diffs |
| `keybindings.go` | Key action parsing |

## Commands

```
aster <file>        View file (auto-detect format)
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
aster jsonl <file|-|+>  Transcripts
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
```

Add new parser: create `parser_xxx.go`, add to `detectParser()` in main.go.

## Release

goreleaser builds cross-platform binaries on tag push.

```bash
git tag v1.0.1
git push origin v1.0.1
# GitHub Actions runs goreleaser -> GitHub Release + Homebrew tap
```

## Testing

```bash
make test
go test -v ./tests/...
```
