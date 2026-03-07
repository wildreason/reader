# Sloan

Manifest right. The human endpoint of the Abe Protocol.

Lap project: wildreason/aster

## Mission

Agents produce hundreds of signals per session. Humans need five. Sloan is the filter — it classifies agent output into human-relevant primitives and renders only what matters. The rendering engine is infrastructure. The product is showing humans the right thing at the right time.

```
sloane (left / parsing)              sloan (right / rendering)
xlsx, docx, zip, pdf                 JSON-LD manifest, agent activity
Unstructured.io wrapper              five primitives classifier
file -> JSON-LD manifest      ->     manifest -> HTML
agent reads this                     human reads this
~/wild/sloane                        ~/wild/aster (becoming ~/wild/sloan)
```

The rendering engine (v1/v2) is complete. Do not extend it. v3 has two tracks: the five primitives classifier (what to show) and the mcp:// fetch layer (where to get it). aster is becoming curl for MCP — point it at any server, get rendered content.

## The Five Primitives

| Primitive | What human sees | Agent signal |
|-----------|----------------|-------------|
| **Decision** | Agent needs your input | AskUserQuestion, permission requests |
| **Artifact** | Agent produced something | Write, create_document, rendered output |
| **Mutation** | Agent changed something | Edit, destructive tools, git push |
| **Threshold** | Cost/time/risk limit hit | Token counts, errors, cost |
| **Status** | Working / done / blocked | Session start/stop, progress |

Everything else is swallowed. This is the product.

## Domain Taste

Sloan is a rendering and classification problem, not a web app problem. The systems worth studying:

**Rendering lineage:** Oberon's text system (content as structured objects, not strings). TeX's box-and-glue model (layout as constraint satisfaction). Plan 9's plumber (content-type routing without file extensions). These teach that rendering is a pipeline — typed input, structured intermediate, formatted output — not template stamping.

**Classification lineage:** Unix `file(1)` (magic bytes, not extensions). Bayesian spam filters (classify by signal density, not keywords). Information retrieval's precision/recall tradeoff — the five primitives optimize for precision (show only what matters) over recall (show everything just in case).

**What this means for code:** Every content type follows `Parse -> []Block -> Classify -> Render`. The `[]Block` pipeline is the architecture. New input sources (hooks, MCP traffic, streaming) enter through new parsers, not new pipelines. New output surfaces (activity feed, manifest viewer) are new renderers on the same blocks. If you're writing code that doesn't touch this pipeline, question whether it belongs here.

## Abe Protocol Position

**Layer:** Manifest (Team Sloane)
**Side:** Right (rendering — human endpoint)
**Counterpart:** sloane (left, parsing — ~/wild/sloane)
**Contract:** JSON-LD manifest schema connects both halves
**Spec:** ~/wild/org/spec.md, Section 3A

## What's Done (engine — do not touch)

- Format-agnostic pipeline: `Parse -> []Block -> Render -> HTML`
- Content types: md, csv, diff, json, jsonl, images, video, contracts, yaml
- Web mode: SSE live reload, syntax highlighting, sortable tables, TOC, search
- TUI mode: terminal rendering with colors and tables
- Static export: `--html` self-contained output
- Browser share: `--share` opens rendered HTML in default browser
- Directory mode: doc index with frontmatter

## What to Build (product)

| # | What | Why |
|---|------|-----|
| 1 | mcp:// registry resolver (LAP-009) | Auto-resolve any public MCP server by name — no config needed |
| 2 | Activity classifier (LAP-006) | Five primitives filter — the product |
| 3 | Hooks integration (LAP-007) | Real-time feed from Claude Code |
| 4 | Manifest renderer | New parser: `parser_manifest.go` for JSON-LD from sloane |
| 5 | Live streaming | `sloan --listen` for push-based agent feeds |

## What's Built (mcp:// fetch layer — LAP-008)

aster fetches content from remote MCP servers via `mcp://` URIs. The fetch layer sits in front of Parse — it resolves the server name, connects via MCP Streamable HTTP, fetches the resource, then feeds content into the existing []Block pipeline.

```
aster mcp://server/resource              # terminal
aster mcp://server/resource --port 8080  # browser
aster mcp://server/resource --html       # static HTML
```

Files: `mcp_uri.go` (parser), `mcp_resolver.go` (name resolution), `mcp_client.go` (MCP transport), `cmd/mcp-demo/` (test server).

Resolution order: `~/.config/aster/servers.json` (local) -> MCP Registry (LAP-009, not yet wired).

**Why this matters:** MCP has naming (registry), discovery (.well-known, in progress), and transport (Streamable HTTP) but no URI scheme connecting them. `mcp://` is the missing glue. 800 of 1000 registered servers have remote endpoints. The registry already returns endpoint URLs in `remotes[]`. Nobody wired it up as a runtime resolver — aster is the first client that does.

## What NOT to Build

- New file format parsers (engine is done)
- TUI improvements (web is the output surface)
- Directory mode features (static site generation is not the product)
- LAP-003/004/005 — engine work, deprioritized
- Auth token management (future — connects to Seal layer / usevault)
- Modifying the MCP spec repo (read-only reference at ~/wild/mcp)

## Architecture

```
Input sources:
  mcp://server/resource          Current: fetch from remote MCP servers (LAP-008)
  Session transcripts (.jsonl)    Current: parse agent activity
  Manifest JSON-LD from sloane    Next: render structured file manifests
  Hooks (PostToolUse, Stop)       Planned: real-time push (LAP-007)
  Stream-JSON pipe (claude -p)    Future: live streaming
     |
     v
[Fetch] -> Parse -> []Block -> Classify (five primitives) -> Render
  |                                                            |
  +-- mcp:// resolve + MCP client (mcp_*.go)                   |
                                                               |
     +-- Web path:  server.go -> formatBlockHTML()              |
     +-- TUI path:  reader.go -> formatter.go                  |
     +-- Static:    RenderStaticHTMLPage                       |
     +-- Activity:  formatActivityFeedHTML() (LAP-006)
```

Block pipeline — all content types produce `[]Block` with typed payloads via `Block.Data`:

| Content type | Parser | Block.Data |
|-------------|--------|------------|
| Markdown | MarkdownParser | - |
| Diffs | DiffParser | - |
| CSV/TSV | CsvParser | *CsvData |
| JSONL | JSONLParser | *TranscriptData |
| Images | ImageParser | *ImageData |
| Video | VideoParser | *VideoData |
| Contracts | ContractParser | *ContractData |

`formatBlockHTML()` dispatches all content types. Add new parser: implement `Parser` interface, add to `detectParser()` in main.go.

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

## Build

```bash
make build     # Build with version injection
make test      # Run tests
make install   # Install to ~/.local/bin
```

Shell alias: `alias as=aster`

## Commands

```
aster <file>               View file (auto-detect format)
aster <file> --share       Open rendered HTML in the browser
aster <file> --port N      Serve as HTML on localhost:N
aster <file> --html        Static self-contained HTML to stdout
aster <dir> --port N       Serve directory as web index
aster session.jsonl        Render agent transcript as activity feed
aster mcp://server/resource  Fetch and render from MCP server
aster pick | p             Pick from recent files
aster latest | l           Open newest file in cwd
```

## Brand

- Fonts: Inter 400/600 (body), JetBrains Mono 400/600 (code)
- Colors: Navy #0A1628, Slate #1E293B, Accent Blue #3B82F6, Surface #F8FAFC
- Semibold for headings (no italic), Accent Blue for interactive only

## Release

```bash
git tag v1.0.4
git push origin v1.0.4
# GitHub Actions -> goreleaser -> GitHub Release + Homebrew tap
```

## Evolution

v1: File -> terminal (done)
v2: File -> web (done)
v3: Agent activity -> web (the product)
v3+: mcp:// -> fetch -> render (curl of MCP — LAP-008 done, LAP-009 next)
