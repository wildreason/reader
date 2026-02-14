# aster

Read any file in the terminal, rendered.

Markdown with colors and tables. Diffs with syntax highlighting. Images inline. JSON, JSONL transcripts, plain text. One binary, auto-detected.

## Install

```bash
brew install wildreason/tap/aster
```

Or build from source:

```bash
git clone https://github.com/wildreason/reader.git
cd reader
make install
```

## Usage

```bash
aster readme.md             # Markdown
aster screenshot.png        # Image (inline in iTerm2, Kitty, WezTerm)
aster changes.patch         # Diff with colors
aster transcript.jsonl      # JSONL conversation viewer
aster data.json             # JSON
aster server.log            # Plain text

aster pick                  # Choose from recently viewed files
aster latest                # Open newest file in current directory
```

Pipe support:

```bash
git diff | aster
curl -s api.example.com | aster
```

## Navigation

```
j / k           Scroll down / up
d / u           Half-page down / up
g / G           Top / bottom
PgDn / PgUp     Full page down / up
q               Quit
```

## Supported formats

| Format | Extensions |
|--------|-----------|
| Markdown | `.md` `.markdown` |
| Plain text | `.txt` `.log` |
| Unified diffs | `.diff` `.patch` |
| JSON | `.json` |
| JSONL transcripts | `.jsonl` |
| Images | `.png` `.jpg` `.gif` `.webp` `.bmp` `.svg` |

Format is auto-detected from the file extension.

## Optional dependencies

Images require [chafa](https://hpjansson.org/chafa/):

```bash
brew install chafa
```

aster auto-detects your terminal and picks the best rendering:
- **iTerm2, WezTerm, Hyper** - iterm inline image protocol (pixel-perfect)
- **Kitty** - kitty graphics protocol
- **Other terminals** - Unicode symbol fallback

## Shell alias

```bash
alias as=aster
```

## License

MIT
