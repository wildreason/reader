package main

import (
	"strings"
	"testing"
)

// --- URL sanitization ---

func TestSanitizeURL_BlocksJavascript(t *testing.T) {
	dangerous := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"JAVASCRIPT:ALERT(1)",
		"javascript:void(0)",
		"  javascript:alert(1)",
	}
	for _, u := range dangerous {
		if result := sanitizeURL(u); result != "#" {
			t.Errorf("sanitizeURL(%q) = %q, want %q", u, result, "#")
		}
	}
}

func TestSanitizeURL_BlocksDataAndVbscript(t *testing.T) {
	dangerous := []string{
		"data:text/html,<script>alert(1)</script>",
		"DATA:text/html,payload",
		"vbscript:MsgBox(1)",
		"VBSCRIPT:payload",
	}
	for _, u := range dangerous {
		if result := sanitizeURL(u); result != "#" {
			t.Errorf("sanitizeURL(%q) = %q, want %q", u, result, "#")
		}
	}
}

func TestSanitizeURL_AllowsSafeSchemes(t *testing.T) {
	safe := []string{
		"https://example.com",
		"http://example.com",
		"mailto:user@example.com",
		"#section-1",
		"/relative/path",
		"../file.md",
	}
	for _, u := range safe {
		if result := sanitizeURL(u); result != u {
			t.Errorf("sanitizeURL(%q) = %q, want unchanged", u, result)
		}
	}
}

// --- Inline HTML rendering ---

func TestProcessInlineHTML_SanitizesLinkHref(t *testing.T) {
	input := `[click me](javascript:alert(1))`
	result := processInlineHTML(input)
	if strings.Contains(result, "javascript:") {
		t.Errorf("processInlineHTML should sanitize javascript: URLs, got: %s", result)
	}
	if !strings.Contains(result, `href="#"`) {
		t.Errorf("processInlineHTML should replace dangerous href with #, got: %s", result)
	}
}

func TestProcessInlineHTML_SanitizesImageSrc(t *testing.T) {
	input := `![xss](javascript:alert(1))`
	result := processInlineHTML(input)
	if strings.Contains(result, "javascript:") {
		t.Errorf("processInlineHTML should sanitize javascript: in image src, got: %s", result)
	}
}

func TestProcessInlineHTML_EscapesHTMLInText(t *testing.T) {
	input := `<script>alert(1)</script>`
	result := processInlineHTML(input)
	if strings.Contains(result, "<script>") {
		t.Errorf("processInlineHTML should escape HTML tags, got: %s", result)
	}
	if !strings.Contains(result, "&lt;script&gt;") {
		t.Errorf("processInlineHTML should produce escaped tags, got: %s", result)
	}
}

func TestProcessInlineHTML_PreservesNormalLinks(t *testing.T) {
	input := `[docs](https://example.com/docs)`
	result := processInlineHTML(input)
	if !strings.Contains(result, `href="https://example.com/docs"`) {
		t.Errorf("processInlineHTML should preserve https links, got: %s", result)
	}
	if !strings.Contains(result, ">docs<") {
		t.Errorf("processInlineHTML should render link text, got: %s", result)
	}
}

// --- Video player HTML ---

func TestVideoPlayerHTML_EscapesTitle(t *testing.T) {
	result := videoPlayerHTML(`<img src=x onerror=alert(1)>.mp4`, "/video", "video/mp4")
	if strings.Contains(result, `<img src=x`) {
		t.Errorf("videoPlayerHTML should escape title, got unescaped HTML in output")
	}
	if !strings.Contains(result, "&lt;img") {
		t.Errorf("videoPlayerHTML should contain escaped title")
	}
}

func TestVideoPlayerHTML_EscapesMime(t *testing.T) {
	result := videoPlayerHTML("test.mp4", "/video", `"><script>alert(1)</script>`)
	if strings.Contains(result, "<script>alert") {
		t.Errorf("videoPlayerHTML should escape mime type")
	}
}

// --- Contract renderer ---

func TestRenderContractHTMLPage_EscapesFrontmatter(t *testing.T) {
	fm := Frontmatter{
		Title: `<script>alert("xss")</script>`,
		Raw: map[string]string{
			"parties":   `<img src=x onerror=alert(1)>`,
			"effective": `2026-01-01" onclick="alert(1)`,
		},
	}
	result := RenderContractHTMLPage(fm.Title, "## 1. Test\n\nBody text.", fm)

	if strings.Contains(result, "<script>alert") {
		t.Errorf("RenderContractHTMLPage should escape title")
	}
	if strings.Contains(result, `<img src=x`) {
		t.Errorf("RenderContractHTMLPage should escape parties")
	}
	// effective date has " which should be escaped to &quot;
	// the raw string should not appear unescaped in an attribute context
	if strings.Contains(result, `"alert(1)`) {
		t.Errorf("RenderContractHTMLPage should escape quotes in effective date")
	}
}

// --- Clause parsing ---

func TestParseContractClauses_Structure(t *testing.T) {
	content := `# Agreement

Preamble text here.

## 1. Definitions

Definition body.

## 2. Scope

Scope body.

### 2.1. Sub-scope

Sub-scope body.`

	preamble, clauses := parseContractClauses(content)

	if !strings.Contains(preamble, "Preamble text") {
		t.Errorf("expected preamble to contain 'Preamble text', got: %s", preamble)
	}
	if len(clauses) != 3 {
		t.Fatalf("expected 3 clauses, got %d", len(clauses))
	}
	if clauses[0].ID != "1" || clauses[0].Title != "Definitions" {
		t.Errorf("clause 0: expected '1. Definitions', got '%s. %s'", clauses[0].ID, clauses[0].Title)
	}
	if clauses[2].ID != "2.1" || clauses[2].Level != 2 {
		t.Errorf("clause 2: expected '2.1' level 2, got '%s' level %d", clauses[2].ID, clauses[2].Level)
	}
}

// --- Server binding ---

func TestServerBindsLocalhost(t *testing.T) {
	// Verify the address format produces localhost binding
	port := 3000
	addr := formatServerAddr(port)
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Errorf("server should bind to 127.0.0.1, got: %s", addr)
	}
}

// --- File size limit ---

func TestMaxFileSizeConstant(t *testing.T) {
	if maxFileSize != 100*1024*1024 {
		t.Errorf("maxFileSize should be 100MB, got %d", maxFileSize)
	}
}
