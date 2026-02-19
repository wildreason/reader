## v1.0.1

- Remove local install step from release target (brew is canonical)

## v1.0.0

First public release.

Terminal file reader. One binary, auto-detected formats.

### Formats

- Markdown with colors, tables, code blocks
- Unified diffs with syntax highlighting
- Images inline (iTerm2, Kitty, WezTerm via chafa)
- JSONL transcript viewer with content type filtering
- JSON
- Plain text

### Features

- Scrollable TUI (j/k, d/u, g/G, PgDn/PgUp)
- Auto-detect format from file extension
- Stdin pipe support
- Recent file picker (`aster pick`)
- Open newest file in cwd (`aster latest`)
- Follow mode for live-updating files
- Type-scoped subcommands (md, img, txt, diff, json, jsonl)

### Install

```
brew install wildreason/tap/aster
```
