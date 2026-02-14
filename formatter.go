package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rivo/tview"
)

// BorderStyle defines visual separation style for blocks
type BorderStyle string

const (
	BorderNone    BorderStyle = "none"
	BorderLeft    BorderStyle = "left"
	BorderMinimal BorderStyle = "minimal"
	BorderBox     BorderStyle = "box"
	BorderDouble  BorderStyle = "double"
	BorderRounded BorderStyle = "rounded"
)

// BorderRenderer handles border formatting logic
type BorderRenderer struct {
	style BorderStyle
}

// NewBorderRenderer creates a renderer for the specified style
func NewBorderRenderer(style BorderStyle) *BorderRenderer {
	return &BorderRenderer{style: style}
}

// RenderLine formats a single line with appropriate border
func (br *BorderRenderer) RenderLine(line string, isEmpty bool) string {
	switch br.style {
	case BorderNone:
		if isEmpty {
			return ""
		}
		return line

	case BorderLeft:
		if isEmpty {
			return "▌"
		}
		return "▌ " + line

	case BorderMinimal:
		if isEmpty {
			return "│"
		}
		return "│ " + line

	case BorderBox, BorderDouble, BorderRounded:
		// For box styles, lines are handled specially in FormatBlockPage
		// This method is used for content lines only
		if isEmpty {
			return ""
		}
		return line

	default:
		return line
	}
}

// RenderBlockStart returns opening border for box-style borders
func (br *BorderRenderer) RenderBlockStart(blockName string, pageInfo string, width int) string {
	header := blockName
	if pageInfo != "" {
		header = header + " " + pageInfo
	}

	// Truncate header if too long
	maxHeaderLen := width - 6 // Account for border chars and padding
	if maxHeaderLen < 10 {
		maxHeaderLen = 10
	}
	if len(header) > maxHeaderLen {
		header = header[:maxHeaderLen-3] + "..."
	}

	switch br.style {
	case BorderBox:
		topLine := "┌" + strings.Repeat("─", width-2) + "┐"
		headerLine := "│ " + header + strings.Repeat(" ", width-4-len(header)) + " │"
		return topLine + "\n" + headerLine

	case BorderDouble:
		topLine := "╔" + strings.Repeat("═", width-2) + "╗"
		headerLine := "║ " + header + strings.Repeat(" ", width-4-len(header)) + " ║"
		return topLine + "\n" + headerLine

	case BorderRounded:
		topLine := "╭" + strings.Repeat("─", width-2) + "╮"
		headerLine := "│ " + header + strings.Repeat(" ", width-4-len(header)) + " │"
		return topLine + "\n" + headerLine

	default:
		return ""
	}
}

// RenderBlockEnd returns closing border for box-style borders
func (br *BorderRenderer) RenderBlockEnd(width int) string {
	switch br.style {
	case BorderBox:
		return "└" + strings.Repeat("─", width-2) + "┘"
	case BorderDouble:
		return "╚" + strings.Repeat("═", width-2) + "╝"
	case BorderRounded:
		return "╰" + strings.Repeat("─", width-2) + "╯"
	default:
		return ""
	}
}

// IsBoxStyle returns true if border style uses top/bottom borders
func (br *BorderRenderer) IsBoxStyle() bool {
	return br.style == BorderBox || br.style == BorderDouble || br.style == BorderRounded
}

// GetContentIndent returns horizontal space consumed by border
func (br *BorderRenderer) GetContentIndent() int {
	switch br.style {
	case BorderNone:
		return 0
	case BorderLeft, BorderMinimal:
		return 2 // "▌ " or "│ "
	case BorderBox, BorderDouble, BorderRounded:
		return 4 // "│ " + " │"
	default:
		return 0
	}
}

// FormatBlockPage renders a specific page of a block with page indicator
func FormatBlockPage(block *Block, pageNum int, termWidth int, borderStyle BorderStyle) string {
	if block == nil {
		return ""
	}

	// Get the page content
	if pageNum < 0 || pageNum >= len(block.Pages) {
		pageNum = 0
	}
	pageContent := block.Pages[pageNum]

	// Determine content type for this specific page
	// PageTypes overrides ContentType when available (for mixed-content blocks)
	pageType := block.ContentType
	if len(block.PageTypes) > pageNum {
		pageType = block.PageTypes[pageNum]
	}

	// Render diff pages with diff formatter
	if pageType == BlockContentDiff {
		// Get filename from PageMeta if available
		filename := ""
		if len(block.PageMeta) > pageNum {
			filename = block.PageMeta[pageNum]
		}
		return FormatDiffPage(block, pageNum, termWidth, filename)
	}

	// Translate ANSI escape codes to tview color tags
	pageContent = tview.TranslateANSI(pageContent)

	// Create border renderer
	renderer := NewBorderRenderer(borderStyle)

	// Adjust content width based on border indent
	contentWidth := termWidth - renderer.GetContentIndent()

	// Render markdown
	rendered := formatMarkdown(pageContent, contentWidth)

	// Determine display name: use page-specific breadcrumb if available
	displayName := block.Name
	if len(block.PageMeta) > pageNum && block.PageMeta[pageNum] != "" {
		displayName = block.PageMeta[pageNum]
	}

	// Add source type prefix with color
	if block.SourceType == SourceChat {
		// Extract just the block number from "block-N" format
		blockNum := strings.TrimPrefix(displayName, "block-")
		if blockNum != displayName { // It was a block-N format
			displayName = "[#b294bb]chat[-] [#808080]" + blockNum + "[-]"
		}
	} else if block.SourceType == SourceShell {
		displayName = "[#99b494]shell[-]"
	}

	// Build output
	var output strings.Builder
	output.WriteString("\n")

	// Box-style borders: render top border and header
	if renderer.IsBoxStyle() {
		pageInfo := ""
		if block.TotalPages > 1 {
			pageInfo = fmt.Sprintf("[%d/%d]", pageNum+1, block.TotalPages)
		}
		start := renderer.RenderBlockStart(displayName, pageInfo, termWidth)
		if start != "" {
			output.WriteString(start)
			output.WriteString("\n")
		}
	} else {
		// Non-box styles: render header (with background highlight only for markdown)
		if block.SourceType == SourceMarkdown {
			// Markdown blocks: gray background highlight
			if block.TotalPages > 1 {
				// Right-align page indicator
				pageIndicator := fmt.Sprintf("[%d/%d]", pageNum+1, block.TotalPages)

				// Calculate spacing to right-align (account for margins)
				spacing := termWidth - len(displayName) - len(pageIndicator) - 4
				if spacing < 1 {
					spacing = 1
				}

				// Build header with gray background, padded to full width
				header := fmt.Sprintf(" %s%s%s ", displayName, strings.Repeat(" ", spacing), pageIndicator)
				// Pad to terminal width for full-width background
				if len(header) < termWidth {
					header = header + strings.Repeat(" ", termWidth-len(header))
				}
				output.WriteString("[white:#333333]" + header + "[-:-:-]")
			} else {
				// Left-align single block name with margin and background
				header := " " + displayName + " "
				if len(header) < termWidth {
					header = header + strings.Repeat(" ", termWidth-len(header))
				}
				output.WriteString("[white:#333333]" + header + "[-:-:-]")
			}
		} else {
			// Non-markdown blocks: no background, simple header
			if block.TotalPages > 1 {
				// Right-align page indicator
				pageIndicator := fmt.Sprintf("[%d/%d]", pageNum+1, block.TotalPages)
				spacing := termWidth - len(displayName) - len(pageIndicator) - 4
				if spacing < 1 {
					spacing = 1
				}
				header := fmt.Sprintf("%s%s%s", displayName, strings.Repeat(" ", spacing), pageIndicator)
				output.WriteString(" " + renderer.RenderLine(header, false))
			} else {
				// Left-align single block name with margin
				output.WriteString(" " + renderer.RenderLine(displayName, false))
			}
		}
		output.WriteString("\n")
		// No extra separator - go directly to content
	}

	// Render content lines
	lines := strings.Split(rendered, "\n")
	for _, line := range lines {
		isEmpty := (line == "")
		if renderer.IsBoxStyle() {
			// For box styles, wrap each line
			if isEmpty {
				output.WriteString("│" + strings.Repeat(" ", termWidth-2) + "│")
			} else {
				// Pad line to fit in box
				if len(line) < contentWidth {
					line = line + strings.Repeat(" ", contentWidth-len(line))
				}
				output.WriteString("│ " + line + " │")
			}
		} else {
			// For other styles, add left margin and use renderer
			if isEmpty {
				output.WriteString(renderer.RenderLine("", isEmpty))
			} else {
				output.WriteString(" " + renderer.RenderLine(line, isEmpty))
			}
		}
		output.WriteString("\n")
	}

	// Box-style borders: render bottom border
	if renderer.IsBoxStyle() {
		end := renderer.RenderBlockEnd(termWidth)
		if end != "" {
			output.WriteString(end)
			output.WriteString("\n")
		}
	} else {
		output.WriteString(renderer.RenderLine("", true))
		output.WriteString("\n")
	}

	return output.String()
}

// FormatBlockPlain renders a block (first page only, for backwards compatibility)
func FormatBlockPlain(block *Block, termWidth int, style string, borderStyle BorderStyle) string {
	return FormatBlockPage(block, 0, termWidth, borderStyle)
}

// FormatDiffPage renders a diff page from a mixed-content block
// Uses same header style as plain pages for consistency
func FormatDiffPage(block *Block, pageNum int, termWidth int, filename string) string {
	// Get the diff content from the page
	if pageNum < 0 || pageNum >= len(block.Pages) {
		return ""
	}
	diffContent := block.Pages[pageNum]

	// Parse hunks from this diff content
	hunks := ParseHunks(diffContent)
	if len(hunks) == 0 {
		return diffContent
	}

	// For mixed-content blocks, we need to find which hunk this page represents
	// Count diff pages before this one to determine hunk index
	hunkIndex := 0
	for i := 0; i < pageNum; i++ {
		if len(block.PageTypes) > i && block.PageTypes[i] == BlockContentDiff {
			// Check if same diff content (same file)
			if len(block.PageMeta) > i && block.PageMeta[i] == filename {
				hunkIndex++
			}
		}
	}

	// Clamp hunk index
	if hunkIndex >= len(hunks) {
		hunkIndex = len(hunks) - 1
	}

	// Create formatter
	formatter := NewDiffFormatter(termWidth)

	// Use provided filename, fallback to extracting from diff
	if filename == "" {
		filename = GetFileFromDiff(diffContent)
	}
	if filename == "" {
		filename = "diff"
	}

	// Format the specific hunk content
	hunkContent := formatter.FormatHunk(hunks[hunkIndex], hunkIndex, len(hunks), filename)

	// Build output with consistent header (same as plain pages)
	var output strings.Builder
	output.WriteString("\n")

	// Header: block-N + filename + [page/total]
	pageIndicator := fmt.Sprintf("[%d/%d]", pageNum+1, block.TotalPages)

	// Extract just filename from path for display
	displayName := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		displayName = filename[idx+1:]
	}

	// Format: block-N  filename  [page/total]
	header := fmt.Sprintf("%s  [green]%s[-]", block.Name, displayName)
	spacing := termWidth - len(block.Name) - len(displayName) - len(pageIndicator) - 8
	if spacing < 1 {
		spacing = 1
	}
	header = fmt.Sprintf(" %s%s%s", header, strings.Repeat(" ", spacing), pageIndicator)
	output.WriteString(header)
	output.WriteString("\n\n")

	// Diff content
	output.WriteString(hunkContent)

	return output.String()
}

// formatMarkdown performs lightweight markdown rendering
// Handles: code blocks, tables, inline code, bold, italic, lists, links
func formatMarkdown(text string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 76 // Default
	}

	lines := strings.Split(text, "\n")
	var result []string
	inCodeBlock := false
	var codeBlockLines []string
	var codeBlockLanguage string
	inTable := false
	var tableLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle tables - detect lines with pipes (must check before code blocks)
		if !inCodeBlock && isTableLine(trimmed) {
			if !inTable {
				inTable = true
				tableLines = []string{line}
			} else {
				tableLines = append(tableLines, line)
			}
			continue
		} else if inTable {
			// End of table - render as table if it fits, otherwise list
			tableResult := renderTable(tableLines, maxWidth)
			if tableResult == nil {
				tableResult = tableToList(tableLines)
			}
			result = append(result, tableResult...)
			inTable = false
			tableLines = nil
			// Fall through to process current line
		}

		// Handle code blocks (```language ... ```)
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				// Starting code block - extract language if present
				codeBlockLanguage = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				codeBlockLines = []string{}
				inCodeBlock = true
			} else {
				// Ending code block - render the code block with wrapper
				codeBlock := renderCodeBlock(codeBlockLines, codeBlockLanguage, maxWidth)
				result = append(result, codeBlock...)
				result = append(result, "") // Empty line after code block
				inCodeBlock = false
				codeBlockLines = nil
			}
			continue
		}

		if inCodeBlock {
			// Collect code block lines
			codeBlockLines = append(codeBlockLines, line)
			continue
		}

		// Process regular markdown line
		processed := processMarkdownLine(line, maxWidth)
		result = append(result, processed...)
	}

	// Handle unclosed code block (edge case)
	if inCodeBlock && len(codeBlockLines) > 0 {
		codeBlock := renderCodeBlock(codeBlockLines, codeBlockLanguage, maxWidth)
		result = append(result, codeBlock...)
	}

	// Handle unclosed table (edge case)
	if inTable && len(tableLines) > 0 {
		tableResult := renderTable(tableLines, maxWidth)
		if tableResult == nil {
			tableResult = tableToList(tableLines)
		}
		result = append(result, tableResult...)
	}

	return strings.Join(result, "\n")
}

// renderCodeBlock renders a code block with visual wrapper
// Detects ASCII art and renders without border to avoid conflicts
func renderCodeBlock(lines []string, language string, maxWidth int) []string {
	if len(lines) == 0 {
		return []string{}
	}

	// ASCII art detection: if content has box-drawing chars, render simply
	if containsBoxDrawing(lines) {
		return renderCodeBlockSimple(lines, language)
	}

	// Normal code: use box border
	return renderCodeBlockBoxed(lines, language, maxWidth)
}

// containsBoxDrawing checks if any line has box-drawing characters
func containsBoxDrawing(lines []string) bool {
	boxChars := "─│┌┐└┘├┤┬┴┼═║╔╗╚╝╠╣╦╩╬╭╮╰╯"
	for _, line := range lines {
		for _, ch := range line {
			if strings.ContainsRune(boxChars, ch) {
				return true
			}
		}
	}
	return false
}

// renderCodeBlockSimple renders without border (for ASCII art)
// Uses gray color to keep visually muted
func renderCodeBlockSimple(lines []string, language string) []string {
	var result []string

	gray := "[#707070]"
	reset := "[-]"

	// Language label if present
	if language != "" {
		result = append(result, gray+language+reset)
	}

	// Simple indent - no borders, no truncation, gray text
	for _, line := range lines {
		result = append(result, gray+"    "+line+reset)
	}

	return result
}

// renderCodeBlockBoxed renders with box-drawing border (for normal code)
// Uses gray color to keep code blocks visually muted
func renderCodeBlockBoxed(lines []string, language string, maxWidth int) []string {
	// Calculate the width of the code block (longest line + padding)
	maxLineLen := 0
	for _, line := range lines {
		if len(line) > maxLineLen {
			maxLineLen = len(line)
		}
	}

	// Limit to maxWidth - 4 (for border characters)
	codeWidth := maxLineLen
	if codeWidth > maxWidth-4 {
		codeWidth = maxWidth - 4
	}

	var result []string

	// Gray color for entire code block
	gray := "[#707070]"
	reset := "[-]"

	// Top border with optional language label
	topBorder := "┌" + strings.Repeat("─", codeWidth+2) + "┐"
	if language != "" {
		label := " " + language + " "
		if len(label) <= codeWidth {
			topBorder = "┌" + label + strings.Repeat("─", codeWidth+2-len(label)) + "┐"
		}
	}
	result = append(result, gray+topBorder+reset)

	// Code lines with side borders
	for _, line := range lines {
		displayLine := line
		if len(displayLine) > codeWidth {
			displayLine = displayLine[:codeWidth]
		}
		padded := displayLine + strings.Repeat(" ", codeWidth-len(displayLine))
		result = append(result, gray+"│ "+padded+" │"+reset)
	}

	// Bottom border
	bottomBorder := "└" + strings.Repeat("─", codeWidth+2) + "┘"
	result = append(result, gray+bottomBorder+reset)

	return result
}

// processMarkdownLine processes a single markdown line
func processMarkdownLine(line string, maxWidth int) []string {
	processed := line
	trimmed := strings.TrimSpace(line)

	// Check for headers first (# ## ###) - process before other formatting
	// TODO: Experiment with header colors - may need adjustment
	if strings.HasPrefix(trimmed, "# ") {
		content := strings.TrimPrefix(trimmed, "# ")
		content = processInlineCode(content)
		content = removeMarkdownBold(content)
		return []string{"[yellow:-:b]" + content + "[-:-:-]"}  // FIX: Yellow may be too bright
	}
	if strings.HasPrefix(trimmed, "## ") {
		content := strings.TrimPrefix(trimmed, "## ")
		content = processInlineCode(content)
		content = removeMarkdownBold(content)
		return []string{"[#87ceeb:-:b]" + content + "[-:-:-]"}  // FIX: Light blue for h2
	}
	if strings.HasPrefix(trimmed, "### ") {
		content := strings.TrimPrefix(trimmed, "### ")
		content = processInlineCode(content)
		content = removeMarkdownBold(content)
		return []string{"[#808080:-:b]" + content + "[-:-:-]"}  // FIX: Gray for h3
	}

	// Process in order: code blocks (already handled), then inline code, links, bold, italic, lists
	// Order matters: process inline code before bold/italic to avoid conflicts

	// Process inline code (`code`) - do this first to protect code from other processing
	processed = processInlineCode(processed)

	// Process links [text](url) -> text (url)
	processed = processLinks(processed)

	// Remove bold (**text** or __text__) - must be before italic
	processed = removeMarkdownBold(processed)

	// Remove italic (*text* or _text_) - after bold to avoid conflicts
	processed = removeMarkdownItalic(processed)

	// Process lists (- item or * item) - after removing bold/italic markers
	processed = processListItems(processed)

	// Let tview handle word wrapping for consistent behavior
	return []string{processed}
}

// removeMarkdownBold removes **text** and __text__ markers and applies bold styling
func removeMarkdownBold(text string) string {
	boldStart := "[#ffd700:-:b]"  // Gold for bold text
	boldEnd := "[-:-:-]"        // Reset all three: foreground, background, flags

	// Use regex for more reliable matching
	// Match **text** (not part of longer sequence) and wrap with bold tags
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	text = boldRegex.ReplaceAllString(text, boldStart+"$1"+boldEnd)

	// Match __text__ (not part of longer sequence) and wrap with bold tags
	boldUnderscoreRegex := regexp.MustCompile(`__([^_]+)__`)
	text = boldUnderscoreRegex.ReplaceAllString(text, boldStart+"$1"+boldEnd)

	return text
}

// removeMarkdownItalic removes *text* and _text_ markers and applies italic styling
func removeMarkdownItalic(text string) string {
	// Use tview regions for italic: [::i]text[::-]
	// This is more reliable than ANSI codes in tview
	italicStart := "[::i]"
	italicEnd := "[::-]"

	// Process single *text* (not **text**)
	// Go regex doesn't support lookbehind, so we use a different approach
	// Match *text* where * is not preceded or followed by another *
	// We'll use a simple state machine approach
	text = removeItalicMarkers(text, '*', italicStart, italicEnd)
	text = removeItalicMarkers(text, '_', italicStart, italicEnd)

	return text
}

// removeItalicMarkers removes single markers (not double) for italic and applies ANSI italic
func removeItalicMarkers(text string, marker byte, italicStart, italicEnd string) string {
	var result strings.Builder
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		if runes[i] == rune(marker) {
			// Check if it's a double marker (bold)
			if i+1 < len(runes) && runes[i+1] == rune(marker) {
				// It's bold, skip both markers (already handled by removeMarkdownBold)
				result.WriteRune(runes[i])
				result.WriteRune(runes[i+1])
				i += 2
				continue
			}

			// Check if it's a single marker (italic) - find the closing marker
			// Look for the next single marker that's not part of a double
			found := false
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == rune(marker) {
					// Check if it's part of a double marker
					if j+1 < len(runes) && runes[j+1] == rune(marker) {
						// This is the start of a double marker, not a closing single
						break
					}
					// Found closing single marker - wrap with ANSI italic codes
					result.WriteString(italicStart)
					result.WriteString(string(runes[i+1 : j]))
					result.WriteString(italicEnd)
					i = j + 1
					found = true
					break
				}
			}

			if !found {
				// No closing marker found, keep the marker
				result.WriteRune(runes[i])
				i++
			}
		} else {
			result.WriteRune(runes[i])
			i++
		}
	}

	return result.String()
}

// processInlineCode formats `code` with tview color tags
func processInlineCode(text string) string {
	// Gray for inline code
	codeRegex := regexp.MustCompile("`([^`]+)`")
	return codeRegex.ReplaceAllString(text, "[#a0a0a0]$1[-]")
}

// processLinks converts [text](url) to blue colored format: [blue]text[white]
// Only shows the link text in blue, hides the URL (still extractable for 'o' key)
// Note: OSC 8 hyperlinks don't work through tview, so we use keyboard shortcut instead
func processLinks(text string) string {
	// Match [text](url) pattern
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	// Replace with blue colored format: [blue]text[white] (URL is hidden but preserved in FullText)
	// This makes links visually distinct and intuitive, like web browsers
	return linkRegex.ReplaceAllString(text, "[blue]$1[white]")
}

// processListItems handles list formatting with colored bullets and consistent indentation
func processListItems(line string) string {
	trimmed := strings.TrimSpace(line)
	leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))

	// TODO: Experiment with indent levels - may need adjustment
	// Base indent for top-level lists, extra for nested
	baseIndent := "  "  // FIX: 2 spaces base indent for all lists
	nestedIndent := "    "  // FIX: 4 spaces for nested lists

	// Check if it's a nested bullet list item (starts with spaces + - or *)
	if leadingSpaces >= 2 && (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
		content := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
		return nestedIndent + "[#808080]-[-] " + content  // FIX: Gray for nested bullets
	}

	// Check if it's a top-level bullet list item (- or *)
	if strings.HasPrefix(trimmed, "- ") {
		content := strings.TrimPrefix(trimmed, "- ")
		return baseIndent + "[cyan]-[-] " + content
	}
	if strings.HasPrefix(trimmed, "* ") {
		content := strings.TrimPrefix(trimmed, "* ")
		return baseIndent + "[cyan]*[-] " + content
	}

	// Check if it's a numbered list (1. 2. 3.)
	if len(trimmed) >= 3 {
		for i := 0; i < len(trimmed) && i < 4; i++ {
			if trimmed[i] >= '0' && trimmed[i] <= '9' {
				continue
			}
			if trimmed[i] == '.' && i > 0 && i+1 < len(trimmed) && trimmed[i+1] == ' ' {
				num := trimmed[:i+1]
				content := trimmed[i+2:]
				return baseIndent + "[yellow]" + num + "[-] " + content
			}
			break
		}
	}

	return line
}

// FormatBlockList renders a list of available blocks
func FormatBlockList(names []string) string {
	if len(names) == 0 {
		return "No blocks found."
	}

	return "Available blocks: " + strings.Join(names, " | ")
}

// FormatHelp returns help text
func FormatHelp() string {
	return `Commands (single-letter preferred):
  j              - next block
  k              - prev block
  l              - list all blocks
  i <name>       - jump to block (fuzzy match)
  h              - show help
  q              - quit

  next           - go to next block
  prev           - go to previous block
  list           - show all available blocks
  jump <name>    - jump to a block
  help           - show this help
  quit / exit    - exit program

Examples:
  > j
  > k
  > l
  > i intro
  > i setup`
}

// FormatError returns a formatted error message
func FormatError(msg string) string {
	return fmt.Sprintf("Error: %s", msg)
}

// FormatNotFound returns a formatted "not found" message with suggestions
func FormatNotFound(query string, availableBlocks []string) string {
	msg := fmt.Sprintf("Block '%s' not found.", query)

	// Try to find close matches
	matches := findClosestMatches(query, availableBlocks, 3)
	if len(matches) > 0 {
		msg += "\nDid you mean: " + strings.Join(matches, ", ") + "?"
	}

	return msg
}

// findClosestMatches finds blocks that contain the query string
func findClosestMatches(query string, blocks []string, limit int) []string {
	query = strings.ToLower(query)
	var matches []string

	for _, block := range blocks {
		if strings.Contains(strings.ToLower(block), query) {
			matches = append(matches, block)
			if len(matches) >= limit {
				break
			}
		}
	}

	return matches
}

// isTableLine checks if a line is part of a markdown table
func isTableLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	// Must have pipes and at least 2 cells
	return strings.Contains(trimmed, "|") && strings.Count(trimmed, "|") >= 2
}

// isTableSeparator checks if a line is a table separator (|---|---|)
func isTableSeparator(line string) bool {
	for _, ch := range line {
		if ch != '-' && ch != ':' && ch != '|' && ch != ' ' {
			return false
		}
	}
	return strings.Contains(line, "-")
}

// parseTableCells extracts cells from a table row
func parseTableCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.Trim(trimmed, "|")

	parts := strings.Split(trimmed, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// renderTable renders a markdown table with box-drawing characters
// Returns nil if the table doesn't fit in maxWidth (caller should fall back to list)
func renderTable(lines []string, maxWidth int) []string {
	if len(lines) < 2 {
		return nil
	}

	// Parse all rows
	var allRows [][]string
	var separatorIdx int = -1
	for i, line := range lines {
		if isTableSeparator(line) {
			separatorIdx = i
			continue
		}
		allRows = append(allRows, parseTableCells(line))
	}

	if len(allRows) < 1 {
		return nil
	}

	// Find number of columns from header
	numCols := len(allRows[0])
	if numCols == 0 {
		return nil
	}

	// Calculate max width per column
	colWidths := make([]int, numCols)
	for _, row := range allRows {
		for c := 0; c < numCols && c < len(row); c++ {
			if len(row[c]) > colWidths[c] {
				colWidths[c] = len(row[c])
			}
		}
	}

	// Calculate total table width: | col1 | col2 | = 1 + (colW+2)*n + 1*(n-1) + 1
	// Each col gets " content " with 1 space padding each side
	totalWidth := 1 // leading │
	for _, w := range colWidths {
		totalWidth += w + 2 + 1 // " content " + │
	}

	if totalWidth > maxWidth {
		return nil // doesn't fit, caller falls back to list
	}

	// Build horizontal lines
	buildHLine := func(left, mid, right, fill string) string {
		var b strings.Builder
		b.WriteString(left)
		for c, w := range colWidths {
			b.WriteString(strings.Repeat(fill, w+2))
			if c < numCols-1 {
				b.WriteString(mid)
			}
		}
		b.WriteString(right)
		return b.String()
	}

	topLine := buildHLine("┌", "┬", "┐", "─")
	midLine := buildHLine("├", "┼", "┤", "─")
	botLine := buildHLine("└", "┴", "┘", "─")

	gray := "[#707070]"

	buildRow := func(cells []string, cellColor string) string {
		var b strings.Builder
		b.WriteString(gray + "│[-]")
		for c := 0; c < numCols; c++ {
			cell := ""
			if c < len(cells) {
				cell = cells[c]
			}
			pad := colWidths[c] - len(cell)
			if cellColor != "" {
				b.WriteString(" " + cellColor + cell + "[-:-:-]" + strings.Repeat(" ", pad) + " " + gray + "│[-]")
			} else {
				b.WriteString(" " + cell + strings.Repeat(" ", pad) + " " + gray + "│[-]")
			}
		}
		return b.String()
	}

	var result []string
	result = append(result, gray+topLine+"[-]")

	// Header row (first row, bold/colored)
	result = append(result, buildRow(allRows[0], "[#87ceeb:-:b]"))

	// Separator after header
	if separatorIdx >= 0 || len(allRows) > 1 {
		result = append(result, gray+midLine+"[-]")
	}

	// Data rows
	for i := 1; i < len(allRows); i++ {
		result = append(result, buildRow(allRows[i], ""))
	}

	result = append(result, gray+botLine+"[-]")
	return result
}

// tableToList converts markdown table to list format
// First column header becomes the label, remaining columns become key-value pairs
func tableToList(lines []string) []string {
	if len(lines) < 2 {
		return lines // Not enough for header + data
	}

	// Parse header row to get column names
	headers := parseTableCells(lines[0])
	if len(headers) == 0 {
		return lines
	}

	// First column header becomes the item label
	itemLabel := headers[0]
	if itemLabel == "" {
		itemLabel = "Item"
	}

	var result []string

	// Process data rows (skip header and separator)
	for i, line := range lines {
		if i == 0 {
			continue // Skip header
		}
		if isTableSeparator(line) {
			continue // Skip separator
		}

		cells := parseTableCells(line)
		if len(cells) == 0 {
			continue
		}

		// First cell with header label (e.g., "Item: listCmd()")
		itemName := cells[0]
		if itemName == "" {
			itemName = "(empty)"
		}
		result = append(result, fmt.Sprintf("[cyan]%s:[-] %s", itemLabel, itemName))

		// Remaining cells become indented key-value pairs
		for j := 1; j < len(cells) && j < len(headers); j++ {
			if cells[j] != "" {
				result = append(result, fmt.Sprintf("    %s: %s", headers[j], cells[j]))
			}
		}

		// Add blank line between items
		result = append(result, "")
	}

	return result
}
