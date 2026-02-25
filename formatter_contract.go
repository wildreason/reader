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

		// Oberon action line - looks like plain text, each word is executable
		clauseID := html.EscapeString(clause.ID)
		sb.WriteString("<div class=\"clause-actions\" id=\"actions-" + clauseID + "\">\n")
		sb.WriteString(fmt.Sprintf("<span class=\"cmd\" data-cmd=\"accept\" data-clause=\"%s\" onclick=\"execCmd(this)\">Accept</span>", clauseID))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd-sep\">|</span>"))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd\" data-cmd=\"reject\" data-clause=\"%s\" onclick=\"execCmd(this)\">Reject</span>", clauseID))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd-sep\">|</span>"))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd\" data-cmd=\"counter\" data-clause=\"%s\" onclick=\"execCmd(this)\">Counter: ___</span>", clauseID))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd-sep\">|</span>"))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd\" data-cmd=\"comment\" data-clause=\"%s\" onclick=\"execCmd(this)\">Comment: ___</span>", clauseID))
		sb.WriteString("\n</div>\n")

		// Inline input (hidden by default, shown for counter/comment)
		sb.WriteString(fmt.Sprintf("<div class=\"clause-input hidden\" id=\"input-%s\">\n", clauseID))
		sb.WriteString(fmt.Sprintf("<input type=\"text\" id=\"input-field-%s\" placeholder=\"Type here...\" onkeydown=\"if(event.key==='Enter')submitInput('%s')\">\n", clauseID, clauseID))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd\" onclick=\"submitInput('%s')\">Submit</span>", clauseID))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd-sep\">|</span>"))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd\" onclick=\"cancelInput('%s')\">Cancel</span>", clauseID))
		sb.WriteString("\n</div>\n")

		// Note display (shown when counter/comment has been submitted)
		sb.WriteString(fmt.Sprintf("<div class=\"clause-note hidden\" id=\"note-%s\">\n", clauseID))
		sb.WriteString(fmt.Sprintf("<span class=\"note-text\" id=\"note-text-%s\"></span>\n", clauseID))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd\" onclick=\"editNote('%s')\">Edit</span>", clauseID))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd-sep\">|</span>"))
		sb.WriteString(fmt.Sprintf("<span class=\"cmd\" onclick=\"clearClause('%s')\">Clear</span>", clauseID))
		sb.WriteString("\n</div>\n")

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
.clause-actions {
  margin-top: 0.75rem;
  padding-top: 0.5rem;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  user-select: none;
}

.cmd {
  color: #94A3B8;
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 2px;
  transition: color 0.15s, background 0.15s;
}

.cmd:hover {
  color: #3B82F6;
  background: #F1F5F9;
}

.cmd.active {
  color: #0A1628;
  font-weight: 600;
}

.cmd-sep {
  color: #E2E8F0;
  margin: 0 0.25rem;
  user-select: none;
}

/* --- Inline input --- */
.clause-input {
  margin-top: 0.5rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
}

.clause-input input {
  flex: 1;
  border: none;
  border-bottom: 1px solid #E2E8F0;
  outline: none;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  padding: 4px 2px;
  color: #0A1628;
  background: transparent;
}

.clause-input input:focus {
  border-bottom-color: #3B82F6;
}

.clause-input input::placeholder {
  color: #CBD5E1;
}

/* --- Note display --- */
.clause-note {
  margin-top: 0.5rem;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
}

.note-text {
  color: #64748B;
  font-style: italic;
}

.note-text::before {
  content: '> ';
  color: #CBD5E1;
}

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

// contractScript returns the Oberon interaction JavaScript
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

var pendingCmd = {};

function execCmd(el) {
  var cmd = el.dataset.cmd;
  var clauseId = el.dataset.clause;

  if (cmd === 'accept' || cmd === 'reject') {
    var state = getState();
    // Toggle: clicking same status clears it
    if (state[clauseId] && state[clauseId].status === cmd) {
      delete state[clauseId];
    } else {
      state[clauseId] = { status: cmd, note: '' };
    }
    saveState(state);
    render();
  } else if (cmd === 'counter' || cmd === 'comment') {
    pendingCmd[clauseId] = cmd;
    var inputDiv = document.getElementById('input-' + clauseId);
    var inputField = document.getElementById('input-field-' + clauseId);
    inputDiv.classList.remove('hidden');
    inputField.placeholder = cmd === 'counter' ? 'Counter-proposal...' : 'Your comment...';
    // Pre-fill if existing note
    var state = getState();
    if (state[clauseId] && state[clauseId].note) {
      inputField.value = state[clauseId].note;
    } else {
      inputField.value = '';
    }
    inputField.focus();
  }
}

function submitInput(clauseId) {
  var inputField = document.getElementById('input-field-' + clauseId);
  var text = inputField.value.trim();
  if (!text) return;

  var cmd = pendingCmd[clauseId] || 'comment';
  var state = getState();
  state[clauseId] = { status: cmd === 'counter' ? 'countered' : 'commented', note: text };
  saveState(state);
  delete pendingCmd[clauseId];
  render();
}

function cancelInput(clauseId) {
  var inputDiv = document.getElementById('input-' + clauseId);
  inputDiv.classList.add('hidden');
  delete pendingCmd[clauseId];
}

function editNote(clauseId) {
  var state = getState();
  var entry = state[clauseId];
  if (!entry) return;
  pendingCmd[clauseId] = entry.status === 'countered' ? 'counter' : 'comment';
  var inputDiv = document.getElementById('input-' + clauseId);
  var inputField = document.getElementById('input-field-' + clauseId);
  var noteDiv = document.getElementById('note-' + clauseId);
  noteDiv.classList.add('hidden');
  inputDiv.classList.remove('hidden');
  inputField.value = entry.note;
  inputField.focus();
}

function clearClause(clauseId) {
  var state = getState();
  delete state[clauseId];
  saveState(state);
  render();
}

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

function render() {
  var state = getState();

  // Count stats
  var counts = { accepted: 0, rejected: 0, countered: 0, commented: 0 };
  var reviewed = 0;
  Object.keys(state).forEach(function(id) {
    var s = state[id].status;
    if (s === 'accept') s = 'accepted';
    if (s === 'reject') s = 'rejected';
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
    var status = entry.status;
    if (status === 'accept') status = 'accepted';
    if (status === 'reject') status = 'rejected';
    var section = document.getElementById('clause-' + id);
    var heading = section ? section.querySelector('.clause-heading') : null;
    var shortTitle = id + '. ' + (heading ? heading.textContent.replace(/\s+/g, ' ').trim().replace(/^\d+\.\s*/, '') : '');
    // Truncate
    if (shortTitle.length > 30) shortTitle = shortTitle.substring(0, 28) + '...';

    var item = document.createElement('a');
    item.className = 'sidebar-item ' + status;
    item.href = '#clause-' + id;
    item.innerHTML = '<span class="item-clause">' + shortTitle + '</span>';
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
    var noteDiv = document.getElementById('note-' + id);
    var noteText = document.getElementById('note-text-' + id);
    var inputDiv = document.getElementById('input-' + id);

    // Reset
    section.classList.remove('state-accepted', 'state-rejected', 'state-countered', 'state-commented');
    badge.textContent = '';
    badge.className = 'clause-badge';
    noteDiv.classList.add('hidden');
    if (!pendingCmd[id]) inputDiv.classList.add('hidden');

    // Highlight active cmd
    var actions = document.getElementById('actions-' + id);
    actions.querySelectorAll('.cmd').forEach(function(cmd) { cmd.classList.remove('active'); });

    if (entry) {
      var status = entry.status;
      if (status === 'accept') status = 'accepted';
      if (status === 'reject') status = 'rejected';

      section.classList.add('state-' + status);
      badge.textContent = status;
      badge.classList.add('badge-' + status);

      // Highlight the active command
      var cmdName = entry.status;
      if (cmdName === 'accepted') cmdName = 'accept';
      if (cmdName === 'rejected') cmdName = 'reject';
      if (cmdName === 'countered') cmdName = 'counter';
      if (cmdName === 'commented') cmdName = 'comment';
      var activeCmd = actions.querySelector('[data-cmd="' + cmdName + '"]');
      if (activeCmd) activeCmd.classList.add('active');

      if (entry.note) {
        noteDiv.classList.remove('hidden');
        noteText.textContent = entry.note;
      }
    }
  });
}

function escapeHtml(text) {
  var div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Keyboard shortcut: navigate clauses with j/k, accept with a, reject with r
var focusedClause = -1;
var clauses = document.querySelectorAll('.clause');

function highlightFocused() {
  clauses.forEach(function(c, i) {
    c.style.outline = (i === focusedClause) ? '2px solid #3B82F6' : 'none';
    c.style.outlineOffset = (i === focusedClause) ? '-2px' : '0';
  });
}

document.addEventListener('keydown', function(e) {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

  var id;
  switch(e.key) {
    case 'j':
      if (focusedClause < 0) focusedClause = 0;
      else focusedClause = Math.min(focusedClause + 1, clauses.length - 1);
      clauses[focusedClause].scrollIntoView({ behavior: 'smooth', block: 'center' });
      highlightFocused();
      break;
    case 'k':
      if (focusedClause < 0) focusedClause = 0;
      else focusedClause = Math.max(focusedClause - 1, 0);
      clauses[focusedClause].scrollIntoView({ behavior: 'smooth', block: 'center' });
      highlightFocused();
      break;
    case 'a':
      if (focusedClause >= 0) {
        id = clauses[focusedClause].dataset.clauseId;
        var state = getState();
        state[id] = { status: 'accept', note: '' };
        saveState(state);
        render();
      }
      break;
    case 'x':
      if (focusedClause >= 0) {
        id = clauses[focusedClause].dataset.clauseId;
        var state = getState();
        state[id] = { status: 'reject', note: '' };
        saveState(state);
        render();
      }
      break;
    case 'c':
      if (focusedClause >= 0) {
        id = clauses[focusedClause].dataset.clauseId;
        pendingCmd[id] = 'counter';
        var inputDiv = document.getElementById('input-' + id);
        var inputField = document.getElementById('input-field-' + id);
        inputDiv.classList.remove('hidden');
        inputField.placeholder = 'Counter-proposal...';
        inputField.value = '';
        inputField.focus();
      }
      break;
    case 'Escape':
      Object.keys(pendingCmd).forEach(function(id) { cancelInput(id); });
      break;
  }
});

// Initial render from localStorage
render();
`, html.EscapeString(title), clauseCount)
}
