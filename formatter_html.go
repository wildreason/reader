package main

import (
	"crypto/sha1"
	"fmt"
	"html"
	"regexp"
	"strings"
)

// RenderHTMLPage renders blocks as a full HTML document with enhanced web features
func RenderHTMLPage(title string, blocks []Block, showLineNums bool) string {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(title)))

	// highlight.js CDN for syntax highlighting
	sb.WriteString("<link rel=\"stylesheet\" href=\"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css\">\n")
	sb.WriteString("<script src=\"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js\"></script>\n")

	sb.WriteString("<style>\n")
	sb.WriteString(cssStyles())
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")

	// Collect headers for TOC
	headers := collectHeaders(blocks)

	// TOC sidebar
	if len(headers) > 1 {
		sb.WriteString("<nav id=\"toc\" class=\"toc\">\n")
		sb.WriteString("<div class=\"toc-toggle\" onclick=\"document.getElementById('toc').classList.toggle('collapsed')\" title=\"Toggle TOC\">&#9776;</div>\n")
		sb.WriteString("<div class=\"toc-content\">\n")
		sb.WriteString(fmt.Sprintf("<div class=\"toc-title\">%s</div>\n", html.EscapeString(title)))
		for _, h := range headers {
			class := "toc-h1"
			if h.level == 2 {
				class = "toc-h2"
			} else if h.level == 3 {
				class = "toc-h3"
			}
			sb.WriteString(fmt.Sprintf("<a class=\"toc-link %s\" href=\"#%s\" data-target=\"%s\">%s</a>\n",
				class, h.id, h.id, html.EscapeString(h.text)))
		}
		sb.WriteString("</div>\n</nav>\n")
	}

	// Main content
	containerClass := "container"
	if len(headers) > 1 {
		containerClass = "container has-toc"
	}
	sb.WriteString(fmt.Sprintf("<main class=\"%s\">\n", containerClass))

	for i := range blocks {
		sb.WriteString(formatBlockHTML(&blocks[i], showLineNums))
	}

	sb.WriteString("</main>\n")

	// Search overlay
	sb.WriteString(searchOverlayHTML())

	sb.WriteString("<script>\n")
	sb.WriteString(enhancedScript())
	sb.WriteString("</script>\n")
	sb.WriteString("</body>\n</html>\n")

	return sb.String()
}

// tocHeader represents a header for the table of contents
type tocHeader struct {
	level int
	text  string
	id    string
}

// collectHeaders scans all blocks for h1/h2/h3 headers
func collectHeaders(blocks []Block) []tocHeader {
	var headers []tocHeader
	for _, block := range blocks {
		for _, page := range block.Pages {
			for _, line := range strings.Split(page, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "### ") {
					text := strings.TrimPrefix(trimmed, "### ")
					headers = append(headers, tocHeader{level: 3, text: text, id: headerID(text)})
				} else if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
					text := strings.TrimPrefix(trimmed, "## ")
					headers = append(headers, tocHeader{level: 2, text: text, id: headerID(text)})
				} else if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
					text := strings.TrimPrefix(trimmed, "# ")
					headers = append(headers, tocHeader{level: 1, text: text, id: headerID(text)})
				}
			}
		}
	}
	return headers
}

// headerID generates a URL-safe anchor ID from header text
func headerID(text string) string {
	// Strip markdown formatting
	text = regexp.MustCompile(`[*_` + "`" + `\[\]()]`).ReplaceAllString(text, "")
	text = strings.ToLower(strings.TrimSpace(text))
	text = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(text, "-")
	text = strings.Trim(text, "-")
	if text == "" {
		text = fmt.Sprintf("h-%x", sha1.Sum([]byte(text)))[:8]
	}
	return text
}

// formatBlockHTML renders a single block with all pages concatenated
func formatBlockHTML(block *Block, showLineNums bool) string {
	var sb strings.Builder

	sb.WriteString("<article class=\"block\">\n")

	// Block header
	displayName := html.EscapeString(block.Name)
	sb.WriteString(fmt.Sprintf("<header class=\"block-header\">%s</header>\n", displayName))

	// Render all pages
	for pageNum := range block.Pages {
		pageContent := block.Pages[pageNum]

		pageType := block.ContentType
		if len(block.PageTypes) > pageNum {
			pageType = block.PageTypes[pageNum]
		}

		if pageType == BlockContentDiff {
			sb.WriteString(formatDiffHTML(pageContent))
		} else {
			sb.WriteString(formatMarkdownHTML(pageContent, block, pageNum, showLineNums))
		}
	}

	sb.WriteString("</article>\n")
	return sb.String()
}

// formatMarkdownHTML renders markdown content as HTML
func formatMarkdownHTML(text string, block *Block, pageNum int, showLineNums bool) string {
	lines := strings.Split(text, "\n")
	var sb strings.Builder
	inCodeBlock := false
	var codeLines []string
	var codeLang string
	inTable := false
	var tableLines []string

	// Determine starting line number for this page
	startLine := 0
	if showLineNums && len(block.PageStartLine) > pageNum {
		startLine = block.PageStartLine[pageNum]
	}

	sb.WriteString("<div class=\"content\">\n")

	flushTable := func() {
		if len(tableLines) > 0 {
			sb.WriteString(renderTableHTML(tableLines))
			tableLines = nil
		}
		inTable = false
	}

	for lineIdx, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Table detection (before code blocks)
		if !inCodeBlock && isTableLine(trimmed) {
			if !inTable {
				inTable = true
				tableLines = []string{line}
			} else {
				tableLines = append(tableLines, line)
			}
			continue
		} else if inTable {
			flushTable()
		}

		// Code fence
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				codeLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				codeLines = []string{}
				inCodeBlock = true
			} else {
				// End code block - with highlight.js class and copy button
				langClass := ""
				if codeLang != "" {
					langClass = fmt.Sprintf(" class=\"language-%s\"", html.EscapeString(codeLang))
				}
				langLabel := ""
				if codeLang != "" {
					langLabel = fmt.Sprintf("<span class=\"code-lang\">%s</span>", html.EscapeString(codeLang))
				}
				sb.WriteString(fmt.Sprintf("<div class=\"code-block\">%s<button class=\"copy-btn\" onclick=\"copyCode(this)\" title=\"Copy\">&#x2398;</button><pre><code%s>", langLabel, langClass))
				for _, cl := range codeLines {
					sb.WriteString(html.EscapeString(cl))
					sb.WriteString("\n")
				}
				sb.WriteString("</code></pre></div>\n")
				inCodeBlock = false
				codeLines = nil
			}
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		// Line number gutter
		lineNumHTML := ""
		if showLineNums && startLine > 0 {
			fileLineNum := startLine + lineIdx
			lineNumHTML = fmt.Sprintf("<span class=\"line-num\">%d</span>", fileLineNum)
		}

		// Headers with anchor IDs
		if strings.HasPrefix(trimmed, "### ") {
			raw := strings.TrimPrefix(trimmed, "### ")
			content := processInlineHTML(raw)
			id := headerID(raw)
			sb.WriteString(fmt.Sprintf("<h3 id=\"%s\">%s<a class=\"anchor\" href=\"#%s\">#</a>%s</h3>\n", id, lineNumHTML, id, content))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			raw := strings.TrimPrefix(trimmed, "## ")
			content := processInlineHTML(raw)
			id := headerID(raw)
			sb.WriteString(fmt.Sprintf("<h2 id=\"%s\">%s<a class=\"anchor\" href=\"#%s\">#</a>%s</h2>\n", id, lineNumHTML, id, content))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			raw := strings.TrimPrefix(trimmed, "# ")
			content := processInlineHTML(raw)
			id := headerID(raw)
			sb.WriteString(fmt.Sprintf("<h1 id=\"%s\">%s<a class=\"anchor\" href=\"#%s\">#</a>%s</h1>\n", id, lineNumHTML, id, content))
			continue
		}

		// Horizontal rule
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			sb.WriteString("<hr>\n")
			continue
		}

		// Empty line
		if trimmed == "" {
			sb.WriteString("<br>\n")
			continue
		}

		// Image: ![alt](url)
		if imgMatch := regexp.MustCompile(`^!\[([^\]]*)\]\(([^)]+)\)$`).FindStringSubmatch(trimmed); imgMatch != nil {
			alt := html.EscapeString(imgMatch[1])
			src := html.EscapeString(imgMatch[2])
			sb.WriteString(fmt.Sprintf("<div class=\"img-wrapper\">%s<img src=\"%s\" alt=\"%s\" loading=\"lazy\" onclick=\"this.classList.toggle('expanded')\"><div class=\"img-caption\">%s</div></div>\n", lineNumHTML, src, alt, alt))
			continue
		}

		// List items
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))
			content := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			content = processInlineHTML(content)
			class := "list-item"
			if leadingSpaces >= 2 {
				class = "list-item nested"
			}
			sb.WriteString(fmt.Sprintf("<div class=\"%s\">%s<span class=\"bullet\">-</span> %s</div>\n", class, lineNumHTML, content))
			continue
		}

		// Numbered list
		if numMatch := regexp.MustCompile(`^(\d+)\.\s+(.+)$`).FindStringSubmatch(trimmed); numMatch != nil {
			content := processInlineHTML(numMatch[2])
			sb.WriteString(fmt.Sprintf("<div class=\"list-item\">%s<span class=\"list-num\">%s.</span> %s</div>\n", lineNumHTML, html.EscapeString(numMatch[1]), content))
			continue
		}

		// Regular paragraph
		content := processInlineHTML(trimmed)
		sb.WriteString(fmt.Sprintf("<p>%s%s</p>\n", lineNumHTML, content))
	}

	// Flush unclosed table
	if inTable {
		flushTable()
	}

	// Flush unclosed code block
	if inCodeBlock && len(codeLines) > 0 {
		sb.WriteString("<div class=\"code-block\"><pre><code>")
		for _, cl := range codeLines {
			sb.WriteString(html.EscapeString(cl))
			sb.WriteString("\n")
		}
		sb.WriteString("</code></pre></div>\n")
	}

	sb.WriteString("</div>\n")
	return sb.String()
}

// processInlineHTML handles inline markdown: bold, italic, code, links, images
func processInlineHTML(text string) string {
	escaped := html.EscapeString(text)

	// Inline code: `code`
	codeRe := regexp.MustCompile("`([^`]+)`")
	escaped = codeRe.ReplaceAllString(escaped, "<code class=\"inline\">$1</code>")

	// Inline images: ![alt](url)
	imgRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	escaped = imgRe.ReplaceAllString(escaped, `<img class="inline-img" src="$2" alt="$1" loading="lazy">`)

	// Bold: **text**
	boldRe := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	escaped = boldRe.ReplaceAllString(escaped, "<strong>$1</strong>")

	// Bold: __text__
	boldUnderRe := regexp.MustCompile(`__([^_]+)__`)
	escaped = boldUnderRe.ReplaceAllString(escaped, "<strong>$1</strong>")

	// Italic: *text* (not **)
	italicRe := regexp.MustCompile(`\*([^*]+)\*`)
	escaped = italicRe.ReplaceAllString(escaped, "<em>$1</em>")

	// Links: [text](url) -- open in new tab with external icon
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	escaped = linkRe.ReplaceAllString(escaped, `<a href="$2" target="_blank" rel="noopener" title="$2">$1<span class="ext-icon">&#x2197;</span></a>`)

	return escaped
}

// formatDiffHTML renders diff content with side-by-side view, collapsible hunks, and word-level highlighting
func formatDiffHTML(content string) string {
	hunks := ParseHunks(content)
	if len(hunks) == 0 {
		return "<pre>" + html.EscapeString(content) + "</pre>\n"
	}

	var sb strings.Builder
	sb.WriteString("<div class=\"diff\">\n")

	for hunkIdx, hunk := range hunks {
		hunkID := fmt.Sprintf("hunk-%d", hunkIdx)
		sb.WriteString(fmt.Sprintf("<div class=\"diff-hunk\" id=\"%s\">\n", hunkID))
		sb.WriteString(fmt.Sprintf("<div class=\"diff-hunk-header\" onclick=\"toggleHunk('%s-body')\">", hunkID))
		sb.WriteString(fmt.Sprintf("<span class=\"diff-hunk-toggle\">&#x25BC;</span> Hunk %d", hunkIdx+1))
		if hunk.Header != "" {
			sb.WriteString(fmt.Sprintf(" <span class=\"diff-hunk-range\">%s</span>", html.EscapeString(hunk.Header)))
		}
		sb.WriteString("</div>\n")
		sb.WriteString(fmt.Sprintf("<div class=\"diff-hunk-body\" id=\"%s-body\">\n", hunkID))

		// Build paired lines for side-by-side
		sb.WriteString("<table class=\"diff-table\"><colgroup><col class=\"diff-col-num\"><col class=\"diff-col-content\"><col class=\"diff-col-num\"><col class=\"diff-col-content\"></colgroup>\n")

		oldLineNum := hunk.StartOld
		newLineNum := hunk.StartNew

		// Group consecutive removed+added for word-level diff
		i := 0
		for i < len(hunk.Lines) {
			line := hunk.Lines[i]

			if line.Type == DiffContext {
				sb.WriteString(fmt.Sprintf("<tr class=\"diff-row-context\"><td class=\"diff-num\">%d</td><td class=\"diff-code\"> %s</td><td class=\"diff-num\">%d</td><td class=\"diff-code\"> %s</td></tr>\n",
					oldLineNum, html.EscapeString(line.Content), newLineNum, html.EscapeString(line.Content)))
				oldLineNum++
				newLineNum++
				i++
				continue
			}

			// Collect consecutive removed lines
			var removed []DiffLine
			for i < len(hunk.Lines) && hunk.Lines[i].Type == DiffRemoved {
				removed = append(removed, hunk.Lines[i])
				i++
			}
			// Collect consecutive added lines
			var added []DiffLine
			for i < len(hunk.Lines) && hunk.Lines[i].Type == DiffAdded {
				added = append(added, hunk.Lines[i])
				i++
			}

			// Pair them up for side-by-side with word-level diff
			maxPairs := len(removed)
			if len(added) > maxPairs {
				maxPairs = len(added)
			}

			for j := 0; j < maxPairs; j++ {
				leftNum := ""
				leftContent := ""
				leftClass := "diff-cell-empty"
				rightNum := ""
				rightContent := ""
				rightClass := "diff-cell-empty"

				if j < len(removed) {
					leftNum = fmt.Sprintf("%d", oldLineNum)
					leftClass = "diff-cell-removed"
					oldLineNum++

					if j < len(added) {
						// Word-level diff between paired lines
						leftHL, rightHL := wordDiffHTML(removed[j].Content, added[j].Content)
						leftContent = leftHL
						rightNum = fmt.Sprintf("%d", newLineNum)
						rightClass = "diff-cell-added"
						rightContent = rightHL
						newLineNum++
					} else {
						leftContent = html.EscapeString(removed[j].Content)
					}
				} else if j < len(added) {
					rightNum = fmt.Sprintf("%d", newLineNum)
					rightClass = "diff-cell-added"
					rightContent = html.EscapeString(added[j].Content)
					newLineNum++
				}

				sb.WriteString(fmt.Sprintf("<tr><td class=\"diff-num %s\">%s</td><td class=\"diff-code %s\">%s</td><td class=\"diff-num %s\">%s</td><td class=\"diff-code %s\">%s</td></tr>\n",
					leftClass, leftNum, leftClass, leftContent,
					rightClass, rightNum, rightClass, rightContent))
			}
		}

		sb.WriteString("</table>\n")
		sb.WriteString("</div>\n</div>\n")
	}

	sb.WriteString("</div>\n")
	return sb.String()
}

// wordDiffHTML computes word-level diff between two lines and returns HTML with highlighted changes
func wordDiffHTML(oldLine, newLine string) (string, string) {
	oldWords := strings.Fields(oldLine)
	newWords := strings.Fields(newLine)

	// Simple LCS-based word diff
	// Build match table
	m := len(oldWords)
	n := len(newWords)

	// LCS length table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldWords[i-1] == newWords[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to find which words match
	oldMatched := make([]bool, m)
	newMatched := make([]bool, n)
	i, j := m, n
	for i > 0 && j > 0 {
		if oldWords[i-1] == newWords[j-1] {
			oldMatched[i-1] = true
			newMatched[j-1] = true
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	// Build HTML with highlights on non-matched words
	var oldHTML, newHTML strings.Builder
	for idx, w := range oldWords {
		if idx > 0 {
			oldHTML.WriteString(" ")
		}
		if oldMatched[idx] {
			oldHTML.WriteString(html.EscapeString(w))
		} else {
			oldHTML.WriteString("<span class=\"diff-word-del\">")
			oldHTML.WriteString(html.EscapeString(w))
			oldHTML.WriteString("</span>")
		}
	}
	for idx, w := range newWords {
		if idx > 0 {
			newHTML.WriteString(" ")
		}
		if newMatched[idx] {
			newHTML.WriteString(html.EscapeString(w))
		} else {
			newHTML.WriteString("<span class=\"diff-word-add\">")
			newHTML.WriteString(html.EscapeString(w))
			newHTML.WriteString("</span>")
		}
	}

	return oldHTML.String(), newHTML.String()
}

// renderTableHTML renders markdown table lines as a sortable HTML table with scroll wrapper
func renderTableHTML(lines []string) string {
	if len(lines) < 2 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<div class=\"table-scroll\">\n")
	sb.WriteString("<table class=\"sortable\">\n")

	headerCells := parseTableCells(lines[0])
	sb.WriteString("<thead><tr>")
	for colIdx, cell := range headerCells {
		sb.WriteString(fmt.Sprintf("<th onclick=\"sortTable(this, %d)\" class=\"sortable-th\">%s <span class=\"sort-icon\">&#x25B4;&#x25BE;</span></th>", colIdx, html.EscapeString(cell)))
	}
	sb.WriteString("</tr></thead>\n")

	sb.WriteString("<tbody>\n")
	for i := 1; i < len(lines); i++ {
		if isTableSeparator(lines[i]) {
			continue
		}
		cells := parseTableCells(lines[i])
		sb.WriteString("<tr>")
		for _, cell := range cells {
			sb.WriteString(fmt.Sprintf("<td>%s</td>", processInlineHTML(cell)))
		}
		sb.WriteString("</tr>\n")
	}
	sb.WriteString("</tbody>\n</table>\n</div>\n")

	return sb.String()
}

// searchOverlayHTML returns the search overlay markup
func searchOverlayHTML() string {
	return `<div id="search-overlay" class="search-overlay hidden">
<div class="search-box">
<input id="search-input" type="text" placeholder="Search..." autocomplete="off">
<div class="search-meta"><span id="search-count"></span><span class="search-hint">Esc to close / Enter to navigate</span></div>
</div>
<div id="search-results" class="search-results"></div>
</div>
`
}

// cssStyles returns the full Catppuccin dark theme CSS with all enhancements
func cssStyles() string {
	return `
* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  background: #1e1e2e;
  color: #cdd6f4;
  font-family: 'SF Mono', 'Fira Code', 'JetBrains Mono', 'Cascadia Code', monospace;
  font-size: 14px;
  line-height: 1.6;
}

/* --- Layout with TOC --- */
.container {
  max-width: 900px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}
.container.has-toc {
  margin-left: 280px;
  max-width: 900px;
  padding: 2rem 1.5rem;
}

/* --- TOC sidebar --- */
.toc {
  position: fixed;
  top: 0;
  left: 0;
  width: 260px;
  height: 100vh;
  background: #181825;
  border-right: 1px solid #313244;
  overflow-y: auto;
  padding: 1rem 0;
  z-index: 100;
  transition: transform 0.2s;
}
.toc.collapsed .toc-content { display: none; }
.toc.collapsed { width: 40px; }
.toc.collapsed ~ .container.has-toc { margin-left: 40px; }

.toc-toggle {
  padding: 0.3rem 0.8rem;
  cursor: pointer;
  color: #6c7086;
  font-size: 16px;
}
.toc-toggle:hover { color: #cdd6f4; }

.toc-title {
  padding: 0.3rem 0.8rem 0.6rem;
  color: #f9e2af;
  font-size: 13px;
  font-weight: bold;
  border-bottom: 1px solid #313244;
  margin-bottom: 0.4rem;
}

.toc-link {
  display: block;
  padding: 0.2rem 0.8rem;
  color: #6c7086;
  text-decoration: none;
  font-size: 12px;
  border-left: 2px solid transparent;
  transition: all 0.15s;
}
.toc-link:hover { color: #cdd6f4; background: #1e1e2e; }
.toc-link.active { color: #89b4fa; border-left-color: #89b4fa; background: #1e1e2e; }
.toc-h2 { padding-left: 1.4rem; }
.toc-h3 { padding-left: 2rem; font-size: 11px; }

@media (max-width: 1100px) {
  .toc { transform: translateX(-100%); }
  .toc.open { transform: translateX(0); }
  .toc-toggle { position: fixed; top: 0.5rem; left: 0.5rem; z-index: 101; background: #181825; border-radius: 4px; padding: 0.3rem 0.6rem; }
  .container.has-toc { margin-left: auto; }
}

/* --- Blocks --- */
.block { margin-bottom: 2rem; }

.block-header {
  background: #333333;
  color: #cdd6f4;
  padding: 0.4rem 0.8rem;
  font-size: 13px;
  margin-bottom: 0;
}

.content { padding: 0.5rem 0; }

/* --- Headers with anchors --- */
h1, h2, h3 { position: relative; }
h1 { color: #f9e2af; font-size: 1.4em; margin: 1rem 0 0.5rem; padding-left: 0.8rem; }
h2 { color: #87ceeb; font-size: 1.2em; margin: 1rem 0 0.5rem; padding-left: 0.8rem; }
h3 { color: #808080; font-size: 1.1em; margin: 0.8rem 0 0.4rem; padding-left: 0.8rem; }

.anchor {
  color: #45475a;
  text-decoration: none;
  font-size: 0.7em;
  margin-right: 0.4rem;
  opacity: 0;
  transition: opacity 0.15s;
}
h1:hover .anchor, h2:hover .anchor, h3:hover .anchor { opacity: 1; }
.anchor:hover { color: #89b4fa; }

p { margin: 0.3rem 0; padding-left: 0.8rem; }

strong { color: #ffd700; font-weight: bold; }
em { font-style: italic; }

/* --- Links with external icon --- */
a { color: #89b4fa; text-decoration: none; position: relative; }
a:hover { text-decoration: underline; }
a[target="_blank"] .ext-icon {
  font-size: 0.7em;
  margin-left: 0.15em;
  opacity: 0.5;
  vertical-align: super;
}
a[target="_blank"]:hover .ext-icon { opacity: 1; }
a[target="_blank"]:hover::after {
  content: attr(title);
  position: absolute;
  bottom: 100%;
  left: 0;
  background: #313244;
  color: #cdd6f4;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  font-size: 11px;
  white-space: nowrap;
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  z-index: 50;
  pointer-events: none;
}

code.inline {
  color: #a0a0a0;
  background: #313244;
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-size: 0.9em;
}

/* --- Code blocks with copy button + syntax highlighting --- */
.code-block {
  margin: 0.8rem 0;
  border: 1px solid #707070;
  border-radius: 4px;
  overflow-x: auto;
  position: relative;
}

.code-block .code-lang {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  color: #707070;
  font-size: 0.8em;
  border-bottom: 1px solid #707070;
}

.copy-btn {
  position: absolute;
  top: 0.3rem;
  right: 0.4rem;
  background: #313244;
  color: #6c7086;
  border: 1px solid #45475a;
  border-radius: 4px;
  padding: 0.15rem 0.4rem;
  font-size: 12px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s;
  z-index: 2;
}
.code-block:hover .copy-btn { opacity: 1; }
.copy-btn:hover { color: #cdd6f4; border-color: #6c7086; }
.copy-btn.copied { color: #a6e3a1; }

.code-block pre {
  margin: 0;
  padding: 0.6rem 0.8rem;
  overflow-x: auto;
}

.code-block code {
  font-size: 0.9em;
}
/* Override highlight.js background to match theme */
.code-block pre code.hljs { background: transparent; padding: 0; }

/* --- Diff: side-by-side with word-level highlighting --- */
.diff { margin: 0.5rem 0; font-size: 0.9em; }

.diff-hunk { margin-bottom: 0.5rem; border: 1px solid #313244; border-radius: 4px; overflow: hidden; }

.diff-hunk-header {
  background: #313244;
  padding: 0.3rem 0.6rem;
  cursor: pointer;
  user-select: none;
  font-size: 12px;
  color: #6c7086;
}
.diff-hunk-header:hover { color: #cdd6f4; }
.diff-hunk-toggle { display: inline-block; transition: transform 0.15s; font-size: 10px; margin-right: 0.3rem; }
.diff-hunk.collapsed .diff-hunk-toggle { transform: rotate(-90deg); }
.diff-hunk.collapsed .diff-hunk-body { display: none; }
.diff-hunk-range { color: #45475a; font-size: 11px; }

.diff-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-family: inherit;
}
.diff-col-num { width: 3.5em; }
.diff-col-content { width: calc(50% - 3.5em); }

.diff-table tr { border-bottom: none; }
.diff-num {
  color: #45475a;
  text-align: right;
  padding: 0 0.4rem;
  font-size: 11px;
  user-select: none;
  vertical-align: top;
}
.diff-code {
  padding: 0 0.5rem;
  white-space: pre;
  overflow-x: auto;
  vertical-align: top;
}

.diff-cell-removed { background: rgba(90,45,90,0.3); }
.diff-cell-added { background: rgba(45,90,45,0.3); }
.diff-cell-empty { background: #1e1e2e; }
.diff-row-context td { background: transparent; }
.diff-row-context .diff-code { color: #808080; }

.diff-word-del { background: #5a2d5a; color: #fff; border-radius: 2px; padding: 0 1px; }
.diff-word-add { background: #2d5a2d; color: #fff; border-radius: 2px; padding: 0 1px; }

/* --- Tables: sortable + scroll --- */
.table-scroll {
  overflow-x: auto;
  margin: 0.8rem 0;
  border-radius: 4px;
}

table {
  border-collapse: collapse;
  font-size: 0.9em;
  min-width: 100%;
}
th, td {
  border: 1px solid #707070;
  padding: 0.3rem 0.6rem;
}
th {
  background: #313244;
  color: #87ceeb;
  font-weight: bold;
}
.sortable-th {
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}
.sortable-th:hover { background: #45475a; }
.sort-icon { font-size: 0.7em; color: #45475a; margin-left: 0.3em; }
.sortable-th.asc .sort-icon { color: #89b4fa; }
.sortable-th.desc .sort-icon { color: #89b4fa; }

/* --- Images --- */
.img-wrapper {
  margin: 0.8rem 0;
  padding-left: 0.8rem;
}
.img-wrapper img {
  max-width: 100%;
  border-radius: 4px;
  border: 1px solid #313244;
  cursor: pointer;
  transition: max-width 0.2s;
}
.img-wrapper img.expanded { max-width: none; }
.img-caption {
  color: #6c7086;
  font-size: 11px;
  margin-top: 0.2rem;
}
.inline-img {
  max-height: 1.4em;
  vertical-align: middle;
  border-radius: 2px;
}

/* --- Lists --- */
.list-item {
  padding-left: 1.5rem;
  margin: 0.15rem 0;
}
.list-item.nested { padding-left: 3rem; }
.bullet { color: #89dceb; }
.list-num { color: #f9e2af; }

/* --- Line numbers --- */
.line-num {
  color: #555555;
  display: inline-block;
  min-width: 3em;
  text-align: right;
  padding-right: 0.8em;
  user-select: none;
  font-size: 0.85em;
}

hr {
  border: none;
  border-top: 1px solid #707070;
  margin: 1rem 0;
}

br { display: block; content: ""; margin: 0.2rem 0; }

/* --- Search overlay --- */
.search-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0,0,0,0.6);
  z-index: 200;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 15vh;
}
.search-overlay.hidden { display: none; }

.search-box {
  background: #313244;
  border: 1px solid #45475a;
  border-radius: 8px;
  width: 600px;
  max-width: 90vw;
  padding: 0.6rem 0.8rem;
}
.search-box input {
  width: 100%;
  background: transparent;
  border: none;
  color: #cdd6f4;
  font-family: inherit;
  font-size: 16px;
  outline: none;
}
.search-box input::placeholder { color: #45475a; }
.search-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #45475a;
  margin-top: 0.3rem;
}

.search-results {
  background: #313244;
  border: 1px solid #45475a;
  border-radius: 8px;
  width: 600px;
  max-width: 90vw;
  max-height: 50vh;
  overflow-y: auto;
  margin-top: 0.3rem;
}
.search-results:empty { display: none; }

.search-result {
  padding: 0.4rem 0.8rem;
  cursor: pointer;
  border-bottom: 1px solid #1e1e2e;
  font-size: 13px;
}
.search-result:hover, .search-result.active { background: #45475a; }
.search-result .sr-context { color: #6c7086; font-size: 11px; }
.search-result mark { background: #f9e2af; color: #1e1e2e; border-radius: 2px; padding: 0 2px; }

/* Highlight in page */
.search-highlight { background: rgba(249,226,175,0.3); border-radius: 2px; }
`
}

// enhancedScript returns all JavaScript for the enhanced features
func enhancedScript() string {
	return `
/* --- SSE live reload --- */
var es = new EventSource('/events');
es.onmessage = function(e) { if (e.data === 'reload') location.reload(); };
es.onerror = function() { setTimeout(function() { location.reload(); }, 2000); };

/* --- Syntax highlighting --- */
document.querySelectorAll('.code-block pre code').forEach(function(el) {
  hljs.highlightElement(el);
});

/* --- Copy button --- */
function copyCode(btn) {
  var code = btn.parentElement.querySelector('code');
  var text = code.textContent || code.innerText;
  navigator.clipboard.writeText(text).then(function() {
    btn.classList.add('copied');
    btn.innerHTML = '&#x2713;';
    setTimeout(function() {
      btn.classList.remove('copied');
      btn.innerHTML = '&#x2398;';
    }, 1500);
  });
}

/* --- Table sorting --- */
function sortTable(th, colIdx) {
  var table = th.closest('table');
  var tbody = table.querySelector('tbody');
  var rows = Array.from(tbody.querySelectorAll('tr'));
  var isAsc = th.classList.contains('asc');

  // Reset all headers in this table
  table.querySelectorAll('.sortable-th').forEach(function(h) { h.classList.remove('asc', 'desc'); });

  if (isAsc) {
    th.classList.add('desc');
  } else {
    th.classList.add('asc');
  }

  rows.sort(function(a, b) {
    var aText = (a.children[colIdx] || {}).textContent || '';
    var bText = (b.children[colIdx] || {}).textContent || '';
    var aNum = parseFloat(aText);
    var bNum = parseFloat(bText);
    var cmp;
    if (!isNaN(aNum) && !isNaN(bNum)) {
      cmp = aNum - bNum;
    } else {
      cmp = aText.localeCompare(bText);
    }
    return isAsc ? -cmp : cmp;
  });

  rows.forEach(function(row) { tbody.appendChild(row); });
}

/* --- Collapsible diff hunks --- */
function toggleHunk(id) {
  var body = document.getElementById(id);
  if (body) {
    body.parentElement.classList.toggle('collapsed');
  }
}

/* --- Scroll-spy for TOC --- */
(function() {
  var links = document.querySelectorAll('.toc-link');
  if (links.length === 0) return;

  var targets = [];
  links.forEach(function(link) {
    var id = link.getAttribute('data-target');
    var el = document.getElementById(id);
    if (el) targets.push({ link: link, el: el });
  });

  function updateSpy() {
    var scrollY = window.scrollY + 80;
    var active = null;
    for (var i = targets.length - 1; i >= 0; i--) {
      if (targets[i].el.offsetTop <= scrollY) {
        active = targets[i];
        break;
      }
    }
    links.forEach(function(l) { l.classList.remove('active'); });
    if (active) active.link.classList.add('active');
  }

  window.addEventListener('scroll', updateSpy, { passive: true });
  updateSpy();
})();

/* --- Search --- */
(function() {
  var overlay = document.getElementById('search-overlay');
  var input = document.getElementById('search-input');
  var resultsDiv = document.getElementById('search-results');
  var countSpan = document.getElementById('search-count');
  var activeIdx = -1;
  var results = [];

  // Build searchable index from all text content
  var searchItems = [];
  document.querySelectorAll('.block').forEach(function(block) {
    var header = block.querySelector('.block-header');
    var blockName = header ? header.textContent : '';
    block.querySelectorAll('p, h1, h2, h3, .list-item, .code-block code, td, th').forEach(function(el) {
      var text = el.textContent || '';
      if (text.trim()) {
        searchItems.push({ text: text.trim(), el: el, blockName: blockName });
      }
    });
  });

  function openSearch() {
    overlay.classList.remove('hidden');
    input.value = '';
    input.focus();
    resultsDiv.innerHTML = '';
    countSpan.textContent = '';
    activeIdx = -1;
    clearHighlights();
  }

  function closeSearch() {
    overlay.classList.add('hidden');
    clearHighlights();
  }

  function clearHighlights() {
    document.querySelectorAll('.search-highlight').forEach(function(el) {
      var parent = el.parentNode;
      parent.replaceChild(document.createTextNode(el.textContent), el);
      parent.normalize();
    });
  }

  function doSearch(query) {
    resultsDiv.innerHTML = '';
    results = [];
    activeIdx = -1;
    if (!query || query.length < 2) {
      countSpan.textContent = '';
      return;
    }
    var lower = query.toLowerCase();
    searchItems.forEach(function(item) {
      var idx = item.text.toLowerCase().indexOf(lower);
      if (idx !== -1) {
        results.push(item);
      }
    });

    countSpan.textContent = results.length + ' match' + (results.length !== 1 ? 'es' : '');

    results.slice(0, 50).forEach(function(item, i) {
      var div = document.createElement('div');
      div.className = 'search-result';
      var text = item.text;
      var idx = text.toLowerCase().indexOf(lower);
      var start = Math.max(0, idx - 30);
      var end = Math.min(text.length, idx + query.length + 30);
      var snippet = (start > 0 ? '...' : '') + text.substring(start, idx) +
        '<mark>' + text.substring(idx, idx + query.length) + '</mark>' +
        text.substring(idx + query.length, end) + (end < text.length ? '...' : '');
      div.innerHTML = snippet + '<div class="sr-context">' + (item.blockName || '') + '</div>';
      div.addEventListener('click', function() {
        navigateTo(i);
      });
      resultsDiv.appendChild(div);
    });
  }

  function navigateTo(idx) {
    if (idx < 0 || idx >= results.length) return;
    activeIdx = idx;
    var item = results[idx];
    closeSearch();
    item.el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    item.el.classList.add('search-highlight');
    setTimeout(function() { item.el.classList.remove('search-highlight'); }, 3000);
    // Update active class
    resultsDiv.querySelectorAll('.search-result').forEach(function(el, i) {
      el.classList.toggle('active', i === idx);
    });
  }

  if (input) {
    input.addEventListener('input', function() {
      doSearch(input.value);
    });
    input.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') {
        closeSearch();
      } else if (e.key === 'Enter') {
        if (results.length > 0) {
          navigateTo(activeIdx < 0 ? 0 : activeIdx);
        }
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        if (results.length > 0) {
          activeIdx = (activeIdx + 1) % Math.min(results.length, 50);
          resultsDiv.querySelectorAll('.search-result').forEach(function(el, i) {
            el.classList.toggle('active', i === activeIdx);
            if (i === activeIdx) el.scrollIntoView({ block: 'nearest' });
          });
        }
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        if (results.length > 0) {
          activeIdx = activeIdx <= 0 ? Math.min(results.length, 50) - 1 : activeIdx - 1;
          resultsDiv.querySelectorAll('.search-result').forEach(function(el, i) {
            el.classList.toggle('active', i === activeIdx);
            if (i === activeIdx) el.scrollIntoView({ block: 'nearest' });
          });
        }
      }
    });
  }

  // Global keyboard: / or Ctrl+K to open search, Escape to close
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && !overlay.classList.contains('hidden')) {
      closeSearch();
      return;
    }
    if (overlay.classList.contains('hidden') && (e.key === '/' || (e.ctrlKey && e.key === 'k'))) {
      if (document.activeElement && (document.activeElement.tagName === 'INPUT' || document.activeElement.tagName === 'TEXTAREA')) return;
      e.preventDefault();
      openSearch();
    }
  });
})();
`
}
