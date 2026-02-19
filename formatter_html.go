package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// RenderHTMLPage renders blocks as a full HTML document
func RenderHTMLPage(title string, blocks []Block, showLineNums bool) string {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("<style>\n")
	sb.WriteString(cssStyles())
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")
	sb.WriteString("<div class=\"container\">\n")

	for i := range blocks {
		sb.WriteString(formatBlockHTML(&blocks[i], showLineNums))
	}

	sb.WriteString("</div>\n")
	sb.WriteString("<script>\n")
	sb.WriteString(sseScript())
	sb.WriteString("</script>\n")
	sb.WriteString("</body>\n</html>\n")

	return sb.String()
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
// Parallel to formatMarkdown() but emits HTML tags instead of tview tags
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
			// Fall through to process current line
		}

		// Code fence
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				codeLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				codeLines = []string{}
				inCodeBlock = true
			} else {
				// End code block
				langAttr := ""
				if codeLang != "" {
					langAttr = fmt.Sprintf(" data-lang=\"%s\"", html.EscapeString(codeLang))
				}
				langLabel := ""
				if codeLang != "" {
					langLabel = fmt.Sprintf("<span class=\"code-lang\">%s</span>", html.EscapeString(codeLang))
				}
				sb.WriteString(fmt.Sprintf("<div class=\"code-block\"%s>%s<pre><code>", langAttr, langLabel))
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

		// Headers
		if strings.HasPrefix(trimmed, "### ") {
			content := processInlineHTML(strings.TrimPrefix(trimmed, "### "))
			sb.WriteString(fmt.Sprintf("<h3>%s%s</h3>\n", lineNumHTML, content))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			content := processInlineHTML(strings.TrimPrefix(trimmed, "## "))
			sb.WriteString(fmt.Sprintf("<h2>%s%s</h2>\n", lineNumHTML, content))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			content := processInlineHTML(strings.TrimPrefix(trimmed, "# "))
			sb.WriteString(fmt.Sprintf("<h1>%s%s</h1>\n", lineNumHTML, content))
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

// processInlineHTML handles inline markdown: bold, italic, code, links
func processInlineHTML(text string) string {
	escaped := html.EscapeString(text)

	// Inline code: `code`
	codeRe := regexp.MustCompile("`([^`]+)`")
	escaped = codeRe.ReplaceAllString(escaped, "<code class=\"inline\">$1</code>")

	// Bold: **text**
	boldRe := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	escaped = boldRe.ReplaceAllString(escaped, "<strong>$1</strong>")

	// Bold: __text__
	boldUnderRe := regexp.MustCompile(`__([^_]+)__`)
	escaped = boldUnderRe.ReplaceAllString(escaped, "<strong>$1</strong>")

	// Italic: *text* (not **)
	italicRe := regexp.MustCompile(`\*([^*]+)\*`)
	escaped = italicRe.ReplaceAllString(escaped, "<em>$1</em>")

	// Links: [text](url) -- url was escaped, unescape parens for href
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	escaped = linkRe.ReplaceAllString(escaped, `<a href="$2">$1</a>`)

	return escaped
}

// formatDiffHTML renders diff content as HTML
func formatDiffHTML(content string) string {
	hunks := ParseHunks(content)
	if len(hunks) == 0 {
		return "<pre>" + html.EscapeString(content) + "</pre>\n"
	}

	var sb strings.Builder
	sb.WriteString("<div class=\"diff\">\n")

	for _, hunk := range hunks {
		for _, line := range hunk.Lines {
			escaped := html.EscapeString(line.Content)
			switch line.Type {
			case DiffAdded:
				sb.WriteString(fmt.Sprintf("<div class=\"diff-added\">+%s</div>\n", escaped))
			case DiffRemoved:
				sb.WriteString(fmt.Sprintf("<div class=\"diff-removed\">-%s</div>\n", escaped))
			default:
				sb.WriteString(fmt.Sprintf("<div class=\"diff-context\"> %s</div>\n", escaped))
			}
		}
	}

	sb.WriteString("</div>\n")
	return sb.String()
}

// renderTableHTML renders markdown table lines as an HTML table
func renderTableHTML(lines []string) string {
	if len(lines) < 2 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<table>\n")

	headerCells := parseTableCells(lines[0])
	sb.WriteString("<thead><tr>")
	for _, cell := range headerCells {
		sb.WriteString(fmt.Sprintf("<th>%s</th>", html.EscapeString(cell)))
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
	sb.WriteString("</tbody>\n</table>\n")

	return sb.String()
}

// cssStyles returns the Catppuccin dark theme CSS
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

.container {
  max-width: 900px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

.block { margin-bottom: 2rem; }

.block-header {
  background: #333333;
  color: #cdd6f4;
  padding: 0.4rem 0.8rem;
  font-size: 13px;
  margin-bottom: 0;
}

.content { padding: 0.5rem 0; }

h1 { color: #f9e2af; font-size: 1.4em; margin: 1rem 0 0.5rem; }
h2 { color: #87ceeb; font-size: 1.2em; margin: 1rem 0 0.5rem; }
h3 { color: #808080; font-size: 1.1em; margin: 0.8rem 0 0.4rem; }

p { margin: 0.3rem 0; padding-left: 0.8rem; }

strong { color: #ffd700; font-weight: bold; }
em { font-style: italic; }

a { color: #89b4fa; text-decoration: none; }
a:hover { text-decoration: underline; }

code.inline {
  color: #a0a0a0;
  background: #313244;
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-size: 0.9em;
}

.code-block {
  margin: 0.8rem 0;
  border: 1px solid #707070;
  border-radius: 4px;
  overflow-x: auto;
  position: relative;
}

.code-block .code-lang {
  display: block;
  padding: 0.2rem 0.6rem;
  color: #707070;
  font-size: 0.8em;
  border-bottom: 1px solid #707070;
}

.code-block pre {
  margin: 0;
  padding: 0.6rem 0.8rem;
  overflow-x: auto;
}

.code-block code {
  color: #707070;
  font-size: 0.9em;
}

/* Diff */
.diff { margin: 0.5rem 0; font-size: 0.9em; }
.diff-added {
  background: #2d5a2d;
  color: #fff;
  padding: 0.1rem 0.8rem;
  font-family: inherit;
  white-space: pre;
}
.diff-removed {
  background: #5a2d5a;
  color: #fff;
  padding: 0.1rem 0.8rem;
  font-family: inherit;
  white-space: pre;
}
.diff-context {
  color: #808080;
  padding: 0.1rem 0.8rem;
  font-family: inherit;
  white-space: pre;
}

/* Tables */
table {
  border-collapse: collapse;
  margin: 0.8rem 0;
  font-size: 0.9em;
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

/* Lists */
.list-item {
  padding-left: 1.5rem;
  margin: 0.15rem 0;
}
.list-item.nested { padding-left: 3rem; }
.bullet { color: #89dceb; }
.list-num { color: #f9e2af; }

/* Line numbers */
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
`
}

// sseScript returns the JS for SSE live reload
func sseScript() string {
	return `var es = new EventSource('/events');
es.onmessage = function(e) { if (e.data === 'reload') location.reload(); };
es.onerror = function() { setTimeout(function() { location.reload(); }, 2000); };
`
}
