# Sloan

Manifest right. Human endpoint of the Manifest layer (Team Sloane) in the Abe Protocol.

## Goal

Agents produce structured output. Sloan shows humans only what matters.

## Direction

```
sloane (left / parsing)              sloan (right / rendering)
xlsx, docx, zip, pdf                 JSON-LD manifest, agent activity
Unstructured.io wrapper              five primitives classifier
file -> JSON-LD manifest      ->     manifest -> HTML / TUI
agent reads this                     human reads this
~/wild/sloane                        ~/wild/sloan
```

The rendering engine (v1/v2) is complete. Do not extend it. The product is v3: agent activity rendered through five primitives.

## What's done (engine -- do not touch)

- Format-agnostic pipeline: `Parse -> []Block -> Render -> HTML`
- Content types: md, csv, diff, json, jsonl, images, video, contracts, yaml
- Web mode: SSE live reload, syntax highlighting, sortable tables, TOC, search
- TUI mode: terminal rendering with colors and tables
- Directory mode: doc index with frontmatter
- Static export: `--html` self-contained output

## What to build (product)

| Priority | What | Why |
|----------|------|-----|
| 1 | Activity classifier (LAP-006) | Five primitives filter — the product |
| 2 | Hooks integration (LAP-007) | Real-time feed from Claude Code |
| 3 | Manifest renderer | New parser: `parser_manifest.go` for JSON-LD from sloane |
| 4 | Live streaming | `sloan --listen` for push-based agent feeds |

## What NOT to build

- New file format parsers (engine is done — sloane handles parsing)
- TUI improvements (web is the output surface)
- Directory mode features (static site generation is not the product)
- LAP-003 (legacy cleanup), LAP-004 (library API), LAP-005 (public sharing) — engine work, deprioritized
- Generic file rendering polish — it works, leave it

## The five primitives

Agents produce hundreds of records per session. Humans need ~5% of them.

| Primitive | What human sees | Agent signal |
|-----------|----------------|-------------|
| **Decision** | Agent needs your input | AskUserQuestion, permission requests |
| **Artifact** | Agent produced something | Write, create_document, rendered output |
| **Mutation** | Agent changed something | Edit, destructive tools, git push |
| **Threshold** | Cost/time/risk limit hit | Token counts, errors, cost |
| **Status** | Working / done / blocked | Session start/stop, progress |

Everything else is swallowed.

## Abe Protocol position

**Layer:** Manifest (Team Sloane)
**Side:** Right (rendering)
**Counterpart:** sloane (left, parsing — Unstructured.io wrapper)
**Contract:** JSON-LD manifest schema connects both halves

## Build

```bash
make build     # Build with version injection
make test      # Run tests
make install   # Install to ~/.local/bin
```

## Architecture

```
Input sources:
  Session transcripts (.jsonl)    Current: parse agent activity
  Manifest JSON-LD from sloane    Next: render structured file manifests
  Hooks (PostToolUse, Stop)       Planned: real-time push (LAP-007)
  Stream-JSON pipe (claude -p)    Future: live streaming
     |
     v
Parse -> []Block -> Classify (five primitives) -> Render
     |                                              |
     +-- Web path:  server.go -> formatBlockHTML()   |
     +-- TUI path:  reader.go -> formatter.go       |
     +-- Static:    RenderStaticHTMLPage             |
     +-- Activity:  formatActivityFeedHTML() (LAP-006)
```

## Parser Interface

```go
type Parser interface {
    Parse(content string) []Block
    Detect(filePath string) bool
}

type FileParser interface {
    Parser
    ParseFile(filePath string, static bool) ([]Block, error)
}
```

Add manifest parser: create `parser_manifest.go`, implement `Parser`, add to `detectParser()` in main.go.

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry, subcommand routing, auto-detect, flag parsing |
| `data_types.go` | Typed Block payloads |
| `parser.go` | Parser/FileParser interfaces, Block struct |
| `parser_jsonl.go` | JSONLParser (transcripts) |
| `formatter_html.go` | Web rendering: HTML/CSS/JS, brand theme |
| `server.go` | HTTP server, SSE broadcaster, file watcher |
| `content_type.go` | Block content type constants and detection |

## Commands

```
sloan <file>        View file (auto-detect format)
sloan <dir>         List directory docs
sloan <dir> --port  Serve directory as web index
sloan pick | p      Pick from recent files
sloan latest | l    Open newest file in cwd
```

## Brand theme

- Fonts: Inter 400/600 (body), JetBrains Mono 400/600 (code)
- Colors: Navy #0A1628, Slate #1E293B, Accent Blue #3B82F6, Surface #F8FAFC
- Semibold for headings (no italic), Accent Blue for interactive only

## Release

```bash
git tag v1.0.1
git push origin v1.0.1
# GitHub Actions -> goreleaser -> GitHub Release + Homebrew tap
```
