# CLAUDE.md

## Project Overview

**aster** - read any file in the terminal, rendered. Single binary.

```bash
aster file.md           # Markdown with colors and tables
aster photo.png         # Image inline (chafa)
aster changes.diff      # Diff with syntax highlighting
aster data.jsonl        # JSONL transcript viewer
aster pick              # Pick from recent files
aster latest            # Open newest file in cwd
aster -n file.md        # Show source file line numbers
aster file.md --port 3000  # Serve rendered HTML on localhost
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
- `--port N` — Serve rendered file as HTML on localhost:N (SSE live reload on file change)

## Architecture

```
main.go              Args, subcommands, routing, -n/-f flag parsing
     |
detectFileType()     Route by extension: img -> viewImage, text -> parser
     |
detectParser()       Auto-detect: .md .jsonl .diff .txt .json
     |
parser.Parse()       Extract blocks from content (Block.PageStartLine for line mapping)
     |
reader.go            Scrollable TUI (tview), SetLineNumbers config
     |
formatter.go         Render blocks with colors, annotatedLine gutter
```

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry, subcommand routing, auto-detect |
| `viewer_img.go` | Image rendering (chafa/imgcat, iterm/kitty/symbols) |
| `parser.go` | MarkdownParser, Block struct, BlockIndex |
| `parser_jsonl.go` | JSONLParser (transcripts) |
| `parser_diff.go` | DiffParser (unified diffs) |
| `parser_txt.go` | TxtParser (plain text) |
| `parser_todo.go` | TodoParser (JSON todos) |
| `reader.go` | Scrollable TUI viewer |
| `follower.go` | Follow mode (-f), file watching |
| `formatter.go` | Block rendering, markdown, tables, line number gutter |
| `formatter_diff.go` | Diff coloring |
| `formatter_html.go` | Block -> HTML rendering for --port mode |
| `formatter_shell.go` | Shell output styling |
| `server.go` | HTTP server, SSE live reload, watcher integration |
| `content_type.go` | Content type detection |
| `commands.go` | Navigator, command parsing |
| `recent.go` | Recent file history (pick/latest) |
| `context_git.go` | Git context for diffs |
| `render_colors.go` | Color palette (Catppuccin) |
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

## Navigation

- `j/k` - scroll down/up (3 lines)
- `d/u` - half page down/up
- `g/G` - top/bottom
- `PgDn/PgUp` - full page
- `q` - quit

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
