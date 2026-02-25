package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// contractClause represents a numbered section of a contract
type contractClause struct {
	ID    string // "1", "2", "3.1", etc.
	Level int    // 1 = ##, 2 = ###
	Title string // "Definitions", "Scope of Services"
	Body  string // raw markdown body text
}

// parseContractClauses splits markdown content into numbered clauses
func parseContractClauses(content string) (string, []contractClause) {
	var clauses []contractClause
	var current *contractClause
	var bodyLines []string
	var preambleLines []string
	inPreamble := true

	clauseHeadingRe := regexp.MustCompile(`^(#{2,3})\s+(\d+[\d.]*)\.\s+(.*)`)

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		m := clauseHeadingRe.FindStringSubmatch(line)
		if m != nil {
			// Save previous clause
			if current != nil {
				current.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
				clauses = append(clauses, *current)
				bodyLines = nil
			}
			inPreamble = false

			level := 1
			if m[1] == "###" {
				level = 2
			}

			current = &contractClause{
				ID:    m[2],
				Level: level,
				Title: m[3],
			}
		} else if inPreamble {
			preambleLines = append(preambleLines, line)
		} else if current != nil {
			bodyLines = append(bodyLines, line)
		}
	}

	// Last clause
	if current != nil {
		current.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
		clauses = append(clauses, *current)
	}

	return strings.TrimSpace(strings.Join(preambleLines, "\n")), clauses
}

// renderClauseBodyHTML converts clause body markdown to HTML paragraphs
func renderClauseBodyHTML(body string) string {
	var sb strings.Builder
	paragraphs := strings.Split(body, "\n\n")

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// Check for sub-headings (bold lines)
		if strings.HasPrefix(para, "**") && strings.HasSuffix(para, "**") {
			inner := para[2 : len(para)-2]
			sb.WriteString(fmt.Sprintf("<h4 class=\"clause-subheading\">%s</h4>\n", html.EscapeString(inner)))
			continue
		}

		// Handle "**Bold Title.** rest of text" pattern
		processed := processInlineHTML(para)
		sb.WriteString(fmt.Sprintf("<p>%s</p>\n", processed))
	}

	return sb.String()
}

// renderPreambleHTML converts preamble markdown to HTML
func renderPreambleHTML(preamble string) string {
	var sb strings.Builder
	lines := strings.Split(preamble, "\n")

	var currentBlock []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(currentBlock) > 0 {
				text := strings.Join(currentBlock, " ")
				if strings.HasPrefix(text, "# ") {
					title := strings.TrimPrefix(text, "# ")
					sb.WriteString(fmt.Sprintf("<h1 class=\"contract-title\">%s</h1>\n", html.EscapeString(title)))
				} else {
					sb.WriteString(fmt.Sprintf("<p>%s</p>\n", processInlineHTML(text)))
				}
				currentBlock = nil
			}
		} else {
			currentBlock = append(currentBlock, trimmed)
		}
	}
	if len(currentBlock) > 0 {
		text := strings.Join(currentBlock, " ")
		if strings.HasPrefix(text, "# ") {
			title := strings.TrimPrefix(text, "# ")
			sb.WriteString(fmt.Sprintf("<h1 class=\"contract-title\">%s</h1>\n", html.EscapeString(title)))
		} else {
			sb.WriteString(fmt.Sprintf("<p>%s</p>\n", processInlineHTML(text)))
		}
	}

	return sb.String()
}

// RenderContractHTMLPage renders a contract document as a structured read-only page
func RenderContractHTMLPage(title string, content string, fm Frontmatter) string {
	var sb strings.Builder

	preamble, clauses := parseContractClauses(content)

	// Parties from frontmatter
	parties := fm.Raw["parties"]
	effective := fm.Raw["effective"]

	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("<link rel=\"preconnect\" href=\"https://fonts.googleapis.com\">\n")
	sb.WriteString("<link rel=\"preconnect\" href=\"https://fonts.gstatic.com\" crossorigin>\n")
	sb.WriteString("<link rel=\"stylesheet\" href=\"https://fonts.googleapis.com/css2?family=Inter:wght@400;600&family=JetBrains+Mono:wght@400;600&display=swap\">\n")
	sb.WriteString("<style>\n")
	sb.WriteString(contractCSS())
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")

	// TOC sidebar
	sb.WriteString("<nav class=\"contract-toc\">\n")
	sb.WriteString("<div class=\"toc-header\">Contents</div>\n")
	for _, clause := range clauses {
		indent := ""
		if clause.Level == 2 {
			indent = " toc-l2"
		}
		sb.WriteString(fmt.Sprintf("<a class=\"toc-item%s\" href=\"#clause-%s\"><span class=\"toc-num\">%s.</span> %s</a>\n",
			indent, html.EscapeString(clause.ID), html.EscapeString(clause.ID), html.EscapeString(clause.Title)))
	}
	sb.WriteString("</nav>\n")

	// Main contract content
	sb.WriteString("<main class=\"contract\">\n")

	// Contract header
	sb.WriteString("<div class=\"contract-header\">\n")
	if parties != "" {
		sb.WriteString(fmt.Sprintf("<div class=\"contract-parties\">%s</div>\n", html.EscapeString(parties)))
	}
	if effective != "" {
		sb.WriteString(fmt.Sprintf("<div class=\"contract-date\">Effective %s</div>\n", html.EscapeString(effective)))
	}
	sb.WriteString("</div>\n")

	// Preamble
	if preamble != "" {
		sb.WriteString("<div class=\"contract-preamble\">\n")
		sb.WriteString(renderPreambleHTML(preamble))
		sb.WriteString("</div>\n")
	}

	// Clauses
	for _, clause := range clauses {
		levelClass := "clause-l1"
		if clause.Level == 2 {
			levelClass = "clause-l2"
		}

		sb.WriteString(fmt.Sprintf("<section class=\"clause %s\" id=\"clause-%s\">\n",
			levelClass, html.EscapeString(clause.ID)))

		// Clause heading
		tag := "h2"
		if clause.Level == 2 {
			tag = "h3"
		}
		sb.WriteString(fmt.Sprintf("<%s class=\"clause-heading\"><span class=\"clause-num\">%s.</span> %s</%s>\n",
			tag, html.EscapeString(clause.ID), html.EscapeString(clause.Title), tag))

		// Clause body
		sb.WriteString("<div class=\"clause-body\">\n")
		sb.WriteString(renderClauseBodyHTML(clause.Body))
		sb.WriteString("</div>\n")

		sb.WriteString("</section>\n")
	}

	sb.WriteString("</main>\n")

	// SSE live reload
	sb.WriteString(`<script>
var es = new EventSource('/events');
es.onmessage = function(e) { if (e.data === 'reload') location.reload(); };
es.onerror = function() { setTimeout(function() { location.reload(); }, 2000); };
</script>
`)

	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}

// contractCSS returns the contract-specific CSS
func contractCSS() string {
	return `
* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  background: #FFFFFF;
  color: #0A1628;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  font-weight: 400;
  font-size: 16px;
  line-height: 1.8;
  -webkit-font-smoothing: antialiased;
}

/* --- TOC sidebar --- */
.contract-toc {
  position: fixed;
  top: 0;
  left: 0;
  width: 260px;
  height: 100vh;
  background: #FFFFFF;
  border-right: 1px solid #E2E8F0;
  overflow-y: auto;
  padding: 1.5rem 0;
  z-index: 100;
}

.toc-header {
  padding: 0 1.25rem 1rem;
  font-size: 14px;
  font-weight: 600;
  color: #0A1628;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid #E2E8F0;
  margin-bottom: 0.5rem;
}

.toc-item {
  display: block;
  padding: 0.3rem 1.25rem;
  font-size: 13px;
  color: #64748B;
  text-decoration: none;
  border-left: 3px solid transparent;
  transition: background 0.15s, color 0.15s;
}

.toc-item:hover {
  background: #F8FAFC;
  color: #0A1628;
}

.toc-item.toc-l2 {
  padding-left: 2rem;
  font-size: 12px;
}

.toc-num {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.9em;
  color: #94A3B8;
  margin-right: 0.25rem;
}

/* --- Main content --- */
.contract {
  margin-left: 260px;
  max-width: 740px;
  padding: 2rem 2rem 6rem;
}

.contract-header {
  margin-bottom: 2rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid #E2E8F0;
}

.contract-parties {
  font-size: 13px;
  color: #64748B;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.contract-date {
  font-size: 13px;
  color: #94A3B8;
  margin-top: 0.25rem;
}

.contract-preamble {
  margin-bottom: 2.5rem;
}

.contract-preamble .contract-title {
  font-size: 26px;
  font-weight: 600;
  color: #0A1628;
  margin-bottom: 1.5rem;
  line-height: 1.3;
}

.contract-preamble p {
  margin-bottom: 1rem;
  color: #334155;
  text-align: justify;
}

/* --- Clauses --- */
.clause {
  margin-bottom: 0.5rem;
  padding: 1.25rem 1.5rem;
  border-radius: 2px;
}

.clause-l2 {
  margin-left: 1.5rem;
}

.clause-heading {
  font-size: 20px;
  font-weight: 600;
  color: #0A1628;
  margin-bottom: 0.75rem;
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
}

.clause-l2 .clause-heading {
  font-size: 17px;
}

.clause-num {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.85em;
  color: #64748B;
}

.clause-body p {
  margin-bottom: 0.75rem;
  color: #334155;
  text-align: justify;
}

.clause-body p:last-child {
  margin-bottom: 0;
}

.clause-subheading {
  font-size: 15px;
  font-weight: 600;
  color: #1E293B;
  margin-bottom: 0.5rem;
  margin-top: 0.75rem;
}

/* --- Selection highlight --- */
::selection {
  background: #DBEAFE;
  color: #1E40AF;
}

/* --- Responsive --- */
@media (max-width: 900px) {
  .contract-toc {
    width: 200px;
  }
  .contract {
    margin-left: 200px;
    padding: 1.5rem 1rem;
  }
}

@media (max-width: 640px) {
  .contract-toc {
    position: static;
    width: 100%;
    height: auto;
    max-height: 40vh;
    border-right: none;
    border-bottom: 1px solid #E2E8F0;
  }
  .contract {
    margin-left: 0;
  }
}
`
}
