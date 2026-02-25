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
			// Split on ** to handle "**Title.** Content" pattern
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

// RenderContractHTMLPage renders a contract document with Oberon-style interactive clause actions
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

	// Redline sidebar
	sb.WriteString("<nav class=\"redline-sidebar\">\n")
	sb.WriteString("<div class=\"sidebar-header\">\n")
	sb.WriteString("<div class=\"sidebar-title\">Redline</div>\n")
	sb.WriteString("<div class=\"sidebar-meta\" id=\"sidebar-stats\">0 of " + fmt.Sprintf("%d", len(clauses)) + " reviewed</div>\n")
	sb.WriteString("</div>\n")
	sb.WriteString("<div class=\"sidebar-counts\">\n")
	sb.WriteString("<span class=\"count-pill accepted\" id=\"count-accepted\" title=\"Accepted\">0</span>\n")
	sb.WriteString("<span class=\"count-pill rejected\" id=\"count-rejected\" title=\"Rejected\">0</span>\n")
	sb.WriteString("<span class=\"count-pill countered\" id=\"count-countered\" title=\"Countered\">0</span>\n")
	sb.WriteString("<span class=\"count-pill commented\" id=\"count-commented\" title=\"Commented\">0</span>\n")
	sb.WriteString("</div>\n")
	sb.WriteString("<div class=\"sidebar-list\" id=\"redline-list\"></div>\n")
	sb.WriteString("<div class=\"sidebar-footer\">\n")
	sb.WriteString("<span class=\"action-text\" id=\"export-btn\" onclick=\"exportRedline()\">Export Redline</span>\n")
	sb.WriteString("<span class=\"action-text\" id=\"clear-btn\" onclick=\"clearAll()\">Clear All</span>\n")
	sb.WriteString("</div>\n")
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

		sb.WriteString(fmt.Sprintf("<section class=\"clause %s\" id=\"clause-%s\" data-clause-id=\"%s\">\n",
			levelClass, html.EscapeString(clause.ID), html.EscapeString(clause.ID)))

		// Clause heading
		tag := "h2"
		if clause.Level == 2 {
			tag = "h3"
		}
		sb.WriteString(fmt.Sprintf("<%s class=\"clause-heading\"><span class=\"clause-num\">%s.</span> %s",
			tag, html.EscapeString(clause.ID), html.EscapeString(clause.Title)))
		sb.WriteString(fmt.Sprintf("<span class=\"clause-badge\" id=\"badge-%s\"></span>", html.EscapeString(clause.ID)))
		sb.WriteString(fmt.Sprintf("</%s>\n", tag))

		// Clause body
		sb.WriteString("<div class=\"clause-body\">\n")
		sb.WriteString(renderClauseBodyHTML(clause.Body))
		sb.WriteString("</div>\n")

		// Oberon command line - a blank contenteditable text area.
		// No buttons. Type a command, press Enter. The system interprets.
		clauseID := html.EscapeString(clause.ID)
		sb.WriteString(fmt.Sprintf("<div class=\"cmdline\" contenteditable=\"true\" data-clause=\"%s\" id=\"cmdline-%s\" spellcheck=\"false\"></div>\n", clauseID, clauseID))

		sb.WriteString("</section>\n")
	}

	sb.WriteString("</main>\n")

	// JavaScript
	sb.WriteString("<script>\n")
	sb.WriteString(contractScript(title, len(clauses)))
	sb.WriteString("</script>\n")

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

/* --- Sidebar --- */
.redline-sidebar {
  position: fixed;
  top: 0;
  left: 0;
  width: 260px;
  height: 100vh;
  background: #FFFFFF;
  border-right: 1px solid #E2E8F0;
  display: flex;
  flex-direction: column;
  z-index: 100;
}

.sidebar-header {
  padding: 1.5rem 1.25rem 1rem;
  border-bottom: 1px solid #E2E8F0;
}

.sidebar-title {
  font-size: 14px;
  font-weight: 600;
  color: #0A1628;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.sidebar-meta {
  font-size: 12px;
  color: #64748B;
  margin-top: 0.25rem;
}

.sidebar-counts {
  display: flex;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  border-bottom: 1px solid #E2E8F0;
}

.count-pill {
  font-size: 12px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 10px;
  font-family: 'JetBrains Mono', monospace;
}

.count-pill.accepted { background: #DCFCE7; color: #166534; }
.count-pill.rejected { background: #FEE2E2; color: #991B1B; }
.count-pill.countered { background: #FEF3C7; color: #92400E; }
.count-pill.commented { background: #DBEAFE; color: #1E40AF; }

.sidebar-list {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 0;
}

.sidebar-item {
  display: block;
  padding: 0.4rem 1.25rem;
  font-size: 13px;
  color: #64748B;
  text-decoration: none;
  cursor: pointer;
  border-left: 3px solid transparent;
  transition: background 0.15s, border-color 0.15s;
}

.sidebar-item:hover {
  background: #F8FAFC;
}

.sidebar-item.accepted { border-left-color: #22C55E; color: #166534; }
.sidebar-item.rejected { border-left-color: #EF4444; color: #991B1B; }
.sidebar-item.countered { border-left-color: #F59E0B; color: #92400E; }
.sidebar-item.commented { border-left-color: #3B82F6; color: #1E40AF; }

.sidebar-item .item-clause {
  font-weight: 600;
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
}

.sidebar-item .item-note {
  display: block;
  font-size: 11px;
  color: #94A3B8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
}

.sidebar-footer {
  padding: 1rem 1.25rem;
  border-top: 1px solid #E2E8F0;
  display: flex;
  gap: 1rem;
}

.sidebar-footer .action-text {
  font-size: 12px;
  color: #64748B;
  cursor: pointer;
  font-family: 'JetBrains Mono', monospace;
}

.sidebar-footer .action-text:hover {
  color: #3B82F6;
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
  border-left: 3px solid transparent;
  border-radius: 2px;
  transition: border-color 0.2s, background 0.2s;
}

.clause:hover {
  background: #FAFBFC;
}

.clause.state-accepted {
  border-left-color: #22C55E;
  background: #F0FDF4;
}

.clause.state-rejected {
  border-left-color: #EF4444;
  background: #FEF2F2;
}

.clause.state-countered {
  border-left-color: #F59E0B;
  background: #FFFBEB;
}

.clause.state-commented {
  border-left-color: #3B82F6;
  background: #EFF6FF;
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

.clause-badge {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 2px 8px;
  border-radius: 3px;
  margin-left: auto;
  font-family: 'JetBrains Mono', monospace;
}

.clause-badge.badge-accepted { background: #DCFCE7; color: #166534; }
.clause-badge.badge-rejected { background: #FEE2E2; color: #991B1B; }
.clause-badge.badge-countered { background: #FEF3C7; color: #92400E; }
.clause-badge.badge-commented { background: #DBEAFE; color: #1E40AF; }

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

.state-rejected .clause-body p {
  text-decoration: line-through;
  text-decoration-color: #FCA5A5;
  color: #94A3B8;
}

/* --- Oberon command line --- */
.cmdline {
  margin-top: 0.75rem;
  padding: 4px 6px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  color: #64748B;
  min-height: 1.6em;
  border-bottom: 1px solid transparent;
  outline: none;
  transition: border-color 0.15s, color 0.15s;
  white-space: pre-wrap;
  word-break: break-word;
}

.cmdline:focus {
  border-bottom-color: #E2E8F0;
  color: #0A1628;
}

/* Placeholder when empty: shows available commands as ghost text */
.cmdline:empty::before {
  content: 'accept  reject  counter: ...  @name ...  clear';
  color: #E2E8F0;
  pointer-events: none;
}

.cmdline:focus:empty::before {
  color: #CBD5E1;
}

/* Command line state colors */
.cmdline-accepted { color: #166534; }
.cmdline-rejected { color: #991B1B; }
.cmdline-countered { color: #92400E; }
.cmdline-commented { color: #1E40AF; }

/* --- Utility --- */
.hidden { display: none !important; }

/* --- Selection highlight --- */
::selection {
  background: #DBEAFE;
  color: #1E40AF;
}

/* --- Responsive --- */
@media (max-width: 900px) {
  .redline-sidebar {
    width: 200px;
  }
  .contract {
    margin-left: 200px;
    padding: 1.5rem 1rem;
  }
}

@media (max-width: 640px) {
  .redline-sidebar {
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

// contractScript returns the Oberon command interpreter JavaScript
func contractScript(title string, clauseCount int) string {
	return fmt.Sprintf(`
var STORAGE_KEY = 'contract:%s';
var CLAUSE_COUNT = %d;

function getState() {
  try {
    var raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch(e) { return {}; }
}

function saveState(state) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}

// --- Command interpreter ---
// Parse free-form text into a structured action.
// No fixed vocabulary. The system figures out what you meant.
function parseCommand(text) {
  text = text.trim();
  if (!text) return null;
  var lower = text.toLowerCase();

  // Accept
  if (/^(accept|a|ok|approve|yes|lgtm|agreed|ack)$/i.test(text)) {
    return { status: 'accepted', note: '' };
  }

  // Reject
  if (/^(reject|r|x|no|decline|nack|disagree|remove)$/i.test(text)) {
    return { status: 'rejected', note: '' };
  }

  // Counter: explicit prefix
  var m = text.match(/^(?:counter|c|revise|propose|amend|change)[\s:]+(.+)/i);
  if (m) {
    return { status: 'countered', note: m[1].trim() };
  }

  // Mention: @name ...
  if (/^@\w+/.test(text)) {
    return { status: 'commented', note: text };
  }

  // Comment: explicit prefix
  m = text.match(/^(?:comment|note|todo|fixme|nb)[\s:]+(.+)/i);
  if (m) {
    return { status: 'commented', note: m[1].trim() };
  }

  // Flag
  if (/^(flag|review|flag for review|needs review|escalate)$/i.test(text)) {
    return { status: 'commented', note: 'flagged for review' };
  }

  // Clear
  if (/^(clear|reset|undo|delete|remove action)$/i.test(text)) {
    return { _clear: true };
  }

  // Anything else: treat as a comment. All text is a valid command.
  return { status: 'commented', note: text };
}

// --- Execute command on a clause ---
function execOnClause(clauseId, text) {
  var result = parseCommand(text);
  var state = getState();

  if (!result) return;

  if (result._clear) {
    delete state[clauseId];
  } else {
    state[clauseId] = result;
  }

  saveState(state);
  render();
}

// --- Enter key executes the command line ---
document.addEventListener('keydown', function(e) {
  if (!e.target.classList.contains('cmdline')) return;

  if (e.key === 'Enter') {
    e.preventDefault();
    var clauseId = e.target.dataset.clause;
    var text = e.target.innerText.trim();
    if (text) {
      // Blur first so render() can update the text content
      e.target.blur();
      execOnClause(clauseId, text);
    }
  }

  // Escape: blur the command line
  if (e.key === 'Escape') {
    e.target.blur();
  }
});

// Prevent pasting rich text into command line
document.addEventListener('paste', function(e) {
  if (!e.target.classList.contains('cmdline')) return;
  e.preventDefault();
  var text = (e.clipboardData || window.clipboardData).getData('text/plain');
  document.execCommand('insertText', false, text);
});

function clearAll() {
  if (!confirm('Clear all redline actions?')) return;
  localStorage.removeItem(STORAGE_KEY);
  render();
}

function exportRedline() {
  var state = getState();
  var lines = ['REDLINE SUMMARY', ''];
  var keys = Object.keys(state).sort(function(a, b) {
    return parseFloat(a) - parseFloat(b);
  });
  keys.forEach(function(id) {
    var entry = state[id];
    var section = document.getElementById('clause-' + id);
    var heading = section ? section.querySelector('.clause-heading') : null;
    var title = heading ? heading.textContent.replace(/\s+/g, ' ').trim() : id;
    var line = entry.status.toUpperCase() + '  ' + title;
    if (entry.note) line += '\n    > ' + entry.note;
    lines.push(line);
  });
  lines.push('');
  lines.push('Reviewed: ' + keys.length + ' of ' + CLAUSE_COUNT + ' clauses');
  var text = lines.join('\n');
  navigator.clipboard.writeText(text).then(function() {
    var btn = document.getElementById('export-btn');
    btn.textContent = 'Copied';
    setTimeout(function() { btn.textContent = 'Export Redline'; }, 1500);
  });
}

function escapeHtml(text) {
  var div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// --- Render: sync DOM with state ---
function render() {
  var state = getState();

  // Count stats
  var counts = { accepted: 0, rejected: 0, countered: 0, commented: 0 };
  var reviewed = 0;
  Object.keys(state).forEach(function(id) {
    var s = state[id].status;
    if (counts[s] !== undefined) counts[s]++;
    reviewed++;
  });

  document.getElementById('count-accepted').textContent = counts.accepted;
  document.getElementById('count-rejected').textContent = counts.rejected;
  document.getElementById('count-countered').textContent = counts.countered;
  document.getElementById('count-commented').textContent = counts.commented;
  document.getElementById('sidebar-stats').textContent = reviewed + ' of ' + CLAUSE_COUNT + ' reviewed';

  // Update sidebar list
  var list = document.getElementById('redline-list');
  list.innerHTML = '';
  var keys = Object.keys(state).sort(function(a, b) {
    return parseFloat(a) - parseFloat(b);
  });
  keys.forEach(function(id) {
    var entry = state[id];
    var section = document.getElementById('clause-' + id);
    var heading = section ? section.querySelector('.clause-heading') : null;
    var shortTitle = id + '. ' + (heading ? heading.textContent.replace(/\s+/g, ' ').trim().replace(/^\d+\.\s*/, '') : '');
    if (shortTitle.length > 30) shortTitle = shortTitle.substring(0, 28) + '...';

    var item = document.createElement('a');
    item.className = 'sidebar-item ' + entry.status;
    item.href = '#clause-' + id;
    item.innerHTML = '<span class="item-clause">' + escapeHtml(shortTitle) + '</span>';
    if (entry.note) {
      item.innerHTML += '<span class="item-note">' + escapeHtml(entry.note) + '</span>';
    }
    list.appendChild(item);
  });

  // Update each clause
  document.querySelectorAll('.clause').forEach(function(section) {
    var id = section.dataset.clauseId;
    var entry = state[id];
    var badge = document.getElementById('badge-' + id);
    var cmdline = document.getElementById('cmdline-' + id);
    var isFocused = document.activeElement === cmdline;

    // Reset visual state
    section.classList.remove('state-accepted', 'state-rejected', 'state-countered', 'state-commented');
    badge.textContent = '';
    badge.className = 'clause-badge';
    cmdline.classList.remove('cmdline-accepted', 'cmdline-rejected', 'cmdline-countered', 'cmdline-commented');

    if (entry) {
      section.classList.add('state-' + entry.status);
      badge.textContent = entry.status;
      badge.classList.add('badge-' + entry.status);
      cmdline.classList.add('cmdline-' + entry.status);

      // Show the executed command as editable text in the command line.
      // Don't overwrite if user is actively editing.
      if (!isFocused) {
        if (entry.note) {
          // Show the original command form so user can edit and re-execute
          if (entry.status === 'countered') {
            cmdline.textContent = 'counter: ' + entry.note;
          } else {
            cmdline.textContent = entry.note;
          }
        } else {
          cmdline.textContent = entry.status === 'accepted' ? 'accept' : 'reject';
        }
      }
    } else {
      // Clear the command line if not focused
      if (!isFocused) {
        cmdline.textContent = '';
      }
    }
  });
}

// --- Keyboard nav: j/k to move between clauses, Enter focuses cmdline ---
var focusedClause = -1;
var allClauses = document.querySelectorAll('.clause');

function highlightFocused() {
  allClauses.forEach(function(c, i) {
    c.style.outline = (i === focusedClause) ? '2px solid #3B82F6' : 'none';
    c.style.outlineOffset = (i === focusedClause) ? '-2px' : '0';
  });
}

document.addEventListener('keydown', function(e) {
  // Skip if inside a command line or input
  if (e.target.classList && e.target.classList.contains('cmdline')) return;
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

  switch(e.key) {
    case 'j':
      if (focusedClause < 0) focusedClause = 0;
      else focusedClause = Math.min(focusedClause + 1, allClauses.length - 1);
      allClauses[focusedClause].scrollIntoView({ behavior: 'smooth', block: 'center' });
      highlightFocused();
      break;
    case 'k':
      if (focusedClause < 0) focusedClause = 0;
      else focusedClause = Math.max(focusedClause - 1, 0);
      allClauses[focusedClause].scrollIntoView({ behavior: 'smooth', block: 'center' });
      highlightFocused();
      break;
    case 'Enter':
      // Focus the command line of the current clause
      if (focusedClause >= 0) {
        e.preventDefault();
        var id = allClauses[focusedClause].dataset.clauseId;
        var cmdline = document.getElementById('cmdline-' + id);
        cmdline.focus();
        // Place cursor at end
        var range = document.createRange();
        range.selectNodeContents(cmdline);
        range.collapse(false);
        var sel = window.getSelection();
        sel.removeAllRanges();
        sel.addRange(range);
      }
      break;
  }
});

// Initial render from localStorage
render();
`, html.EscapeString(title), clauseCount)
}
