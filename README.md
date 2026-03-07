# aster

Read any file, rendered. Terminal or browser.

```bash
brew install wildreason/tap/aster
```

## Usage

```bash
aster readme.md                    # Markdown with colors and tables
aster screenshot.png               # Image inline (iTerm2, Kitty, WezTerm)
aster changes.patch                # Diff with syntax highlighting
aster data.csv                     # CSV as formatted table
aster transcript.jsonl             # JSONL conversation viewer
aster data.json                    # JSON with highlighting
aster server.log                   # Plain text

aster pick                         # Choose from recently viewed files
aster latest                       # Open newest file in cwd
```

Pipe support:

```bash
git diff | aster
curl -s api.example.com | aster
cat log.jsonl | aster -t jsonl
```

## Browser

```bash
aster readme.md --share            # Open rendered HTML in browser
aster readme.md --port 3000        # Serve with live reload
aster data.csv --port 3000         # Sortable, filterable table
aster session.jsonl --port 3000    # Agent transcript as activity feed
aster ~/docs/ --port 8080          # Directory as web index
aster readme.md --html > out.html  # Self-contained HTML to stdout
```

Web features: live reload (SSE), syntax highlighting, copy button on code blocks, sortable tables (numeric-aware), TOC sidebar with scroll-spy, search (`/` or `Ctrl+K`), CSV per-column filters, diff side-by-side with word-level highlighting, video player with speed controls.

## Formats

| Format | Extensions |
|--------|-----------|
| Markdown | `.md` `.markdown` |
| CSV / TSV | `.csv` `.tsv` |
| Unified diffs | `.diff` `.patch` |
| JSON | `.json` |
| JSONL transcripts | `.jsonl` |
| YAML | `.yaml` `.yml` |
| Images | `.png` `.jpg` `.gif` `.webp` `.bmp` `.svg` |
| Video | `.mp4` `.webm` `.mov` |
| Plain text | `.txt` `.log` |

Auto-detected from extension. Override with `-t TYPE`.

## Flags

```
--share      Open rendered HTML in the default browser
--port N     Serve as web page on localhost:N
--html       Export self-contained HTML to stdout
-t TYPE      Force content type (md, json, jsonl, diff, txt, yaml, csv)
-n           Show source line numbers in gutter
-f           Follow mode (watch file for changes)
```

## Navigation

Terminal:

```
j / k           Scroll down / up
d / u           Half-page down / up
g / G           Top / bottom
PgDn / PgUp     Full page down / up
q               Quit
```

## Examples

```bash
# Git diffs in browser
git diff HEAD | aster --share
git diff main..feature | aster --port 3000

# Agent transcripts
aster session.jsonl --share

# Static export
git diff HEAD | aster --html > review.html
aster data.csv --html > table.html

# Directory index
aster ~/notes/ --port 8080
```

## Install

Homebrew:

```bash
brew install wildreason/tap/aster
```

Build from source:

```bash
git clone https://github.com/wildreason/reader.git
cd reader
make install
```

Shell alias: `alias as=aster`

Terminal image rendering requires [chafa](https://hpjansson.org/chafa/): `brew install chafa`

## License

MIT
