package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// Version information (injected at build time)
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// showLineNumbers enables source file line numbers in the gutter (-n flag)
var showLineNumbers bool

// fileType defines a supported file type with its extensions
type fileType struct {
	name       string
	extensions []string
}

var fileTypes = map[string]fileType{
	"md":    {name: "markdown", extensions: []string{".md", ".markdown"}},
	"img":   {name: "image", extensions: imageExtensions},
	"txt":   {name: "text", extensions: []string{".txt", ".log"}},
	"json":  {name: "json", extensions: []string{".json"}},
	"diff":  {name: "diff", extensions: []string{".diff", ".patch"}},
	"jsonl": {name: "jsonl", extensions: []string{".jsonl"}},
}

func detectTerminalWidth() int {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			return w
		}
	}
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if w, err := parsePositiveInt(cols); err == nil {
			return w
		}
	}
	return 80
}

func detectTerminalHeight() int {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if _, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil && h > 0 {
			return h
		}
	}
	if lines := os.Getenv("LINES"); lines != "" {
		if h, err := parsePositiveInt(lines); err == nil {
			return h
		}
	}
	return 24
}

func parsePositiveInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 0, fmt.Errorf("not positive")
	}
	return n, nil
}

// detectFileType returns the type key for a file path based on extension
func detectFileType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	for key, ft := range fileTypes {
		for _, e := range ft.extensions {
			if ext == e {
				return key
			}
		}
	}
	return ""
}

// detectParser selects the appropriate parser based on file extension
func detectParser(filePath string) Parser {
	parsers := []Parser{
		&TodoParser{},
		&DiffParser{},
		&MarkdownParser{},
		&JSONLParser{},
		&TxtParser{},
	}

	for _, parser := range parsers {
		if parser.Detect(filePath) {
			return parser
		}
	}

	return &MarkdownParser{}
}

// detectParserFromContent tries to detect parser type from content (for stdin)
func detectParserFromContent(content string) Parser {
	if DetectBlockContentType(content) == BlockContentDiff {
		return &DiffParser{}
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var testJSON map[string]interface{}
		if err := json.Unmarshal([]byte(line), &testJSON); err == nil {
			return &JSONLParser{}
		}
		break
	}

	return &MarkdownParser{}
}

func hasStdinData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func readStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var builder strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			if line != "" {
				builder.WriteString(line)
			}
			break
		}
		if err != nil {
			return "", err
		}
		builder.WriteString(line)
	}
	return builder.String(), nil
}

func showContentSelector(content string) map[string]bool {
	types := ScanContentTypes(content)
	if len(types) == 0 {
		return map[string]bool{"user": true, "assistant": true}
	}

	fmt.Println("Scanning transcript...")
	fmt.Println()
	fmt.Println("Found content types:")

	for i, ct := range types {
		check := " "
		if ct.Enabled {
			check = "x"
		}
		fmt.Printf("  %d. [%s] %s (%d)\n", i+1, check, ct.Name, ct.Count)
	}

	fmt.Println()
	fmt.Println("Toggle: 1-9 | Confirm: Enter")
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			break
		}

		for _, ch := range input {
			if ch >= '1' && ch <= '9' {
				idx := int(ch - '1')
				if idx < len(types) {
					types[idx].Enabled = !types[idx].Enabled
				}
			}
		}

		fmt.Print("\033[F")
		for range types {
			fmt.Print("\033[F")
		}
		fmt.Print("\033[F")
		fmt.Print("\033[F")

		fmt.Println("Found content types:")
		for i, ct := range types {
			check := " "
			if ct.Enabled {
				check = "x"
			}
			fmt.Printf("  %d. [%s] %s (%d)\n", i+1, check, ct.Name, ct.Count)
		}
		fmt.Println()
		fmt.Println("Toggle: 1-9 | Confirm: Enter")
		fmt.Print("> ")
	}

	filters := make(map[string]bool)
	for _, ct := range types {
		filters[ct.Name] = ct.Enabled
	}

	fmt.Println()
	return filters
}

// expandPath expands ~ and resolves the path
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = strings.Replace(path, "~", home, 1)
		}
	}
	return path
}

// resolveShortcut handles -, +, help, and file arguments for a given type
func resolveShortcut(arg string, exts []string) string {
	switch arg {
	case "-":
		path, err := ShowRecentPicker(exts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return path
	case "+":
		path, err := GetNewestFile(exts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Opening: %s\n", path)
		return path
	case "help":
		return ""
	default:
		return expandPath(arg)
	}
}

// viewFile routes to the correct viewer based on file type
func viewFile(filePath string) {
	if detectFileType(filePath) == "img" {
		AddRecent(filePath)
		viewImage(filePath)
		return
	}
	viewTextFile(filePath, "", false)
}

// viewTextFile reads a file and renders it in the TUI
func viewTextFile(filePath string, forceType string, follow bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not read file '%s': %v\n", filePath, err)
		os.Exit(1)
	}
	fileContent := string(content)
	termWidth := detectTerminalWidth()

	AddRecent(filePath)

	var parser Parser
	var isJSONL bool

	if forceType != "" {
		switch forceType {
		case "md":
			parser = &MarkdownParser{}
		case "jsonl":
			parser = &JSONLParser{}
			isJSONL = true
		case "diff":
			parser = &DiffParser{}
		case "json":
			parser = &TodoParser{}
		case "txt":
			parser = &TxtParser{}
		}
	} else {
		parser = detectParser(filePath)
		_, isJSONL = parser.(*JSONLParser)
	}

	if follow {
		runFollowMode(filePath, fileContent, isJSONL, termWidth, "auto", BorderNone)
		return
	}

	var blocks []Block
	if isJSONL {
		jsonlParser := &JSONLParser{}
		filters := showContentSelector(fileContent)
		jsonlParser.Filters = filters
		blocks = jsonlParser.Parse(fileContent)
	} else if mdParser, ok := parser.(*MarkdownParser); ok {
		termHeight := detectTerminalHeight()
		blocks = mdParser.ParseContinuous(fileContent, termHeight)
	} else {
		blocks = parser.Parse(fileContent)
	}

	runReaderMode(blocks, filePath, termWidth, "auto", BorderNone)
}

// viewStdinContent renders stdin content
func viewStdinContent(content string, forceType string) {
	termWidth := detectTerminalWidth()

	var parser Parser
	var isJSONL bool

	if forceType != "" {
		switch forceType {
		case "md":
			parser = &MarkdownParser{}
		case "jsonl":
			parser = &JSONLParser{}
			isJSONL = true
		case "diff":
			parser = &DiffParser{}
		case "json":
			parser = &TodoParser{}
		case "txt":
			parser = &TxtParser{}
		default:
			parser = &MarkdownParser{}
		}
	} else {
		parser = detectParserFromContent(content)
		_, isJSONL = parser.(*JSONLParser)
	}

	var blocks []Block
	if isJSONL {
		jsonlParser := &JSONLParser{}
		filters := showContentSelector(content)
		jsonlParser.Filters = filters
		blocks = jsonlParser.Parse(content)
	} else if mdParser, ok := parser.(*MarkdownParser); ok {
		termHeight := detectTerminalHeight()
		blocks = mdParser.ParseContinuous(content, termHeight)
	} else {
		blocks = parser.Parse(content)
	}

	runReaderMode(blocks, "stdin", termWidth, "auto", BorderNone)
}

func printUsage() {
	w := os.Stderr
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Read any file in the terminal, rendered.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  aster <file>          View file (auto-detect format)")
	fmt.Fprintln(w, "  aster pick            Pick from recent files")
	fmt.Fprintln(w, "  aster latest          Open newest file in current directory")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Supported formats:")
	fmt.Fprintln(w, "  Markdown        .md .markdown")
	fmt.Fprintln(w, "  Plain text      .txt .log")
	fmt.Fprintln(w, "  Unified diffs   .diff .patch")
	fmt.Fprintln(w, "  JSON            .json")
	fmt.Fprintln(w, "  Transcripts     .jsonl")
	fmt.Fprintln(w, "  Images          .png .jpg .gif .webp .bmp .svg")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Navigation:")
	fmt.Fprintln(w, "  j / k             Scroll down / up")
	fmt.Fprintln(w, "  d / u             Half-page down / up")
	fmt.Fprintln(w, "  g / G             Top / bottom")
	fmt.Fprintln(w, "  PgDn / PgUp       Full page down / up")
	fmt.Fprintln(w, "  q                 Quit")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  aster readme.md               View markdown with colors and tables")
	fmt.Fprintln(w, "  aster screenshot.png          Render image inline")
	fmt.Fprintln(w, "  aster changes.patch           View diff with syntax highlighting")
	fmt.Fprintln(w, "  aster pick                    Choose from recently viewed files")
	fmt.Fprintln(w, "  aster latest                  Open the newest file in cwd")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Images require chafa (brew install chafa).")
	fmt.Fprintln(w)
}

func main() {
	// Parse -n flag early (before other arg processing)
	var cleanArgs []string
	for _, arg := range os.Args[1:] {
		if arg == "-n" {
			showLineNumbers = true
		} else {
			cleanArgs = append(cleanArgs, arg)
		}
	}
	os.Args = append([]string{os.Args[0]}, cleanArgs...)

	// Check for subcommand or shortcut as first arg
	if len(os.Args) >= 2 {
		first := os.Args[1]

		switch {
		case first == "help" || first == "--help":
			printUsage()
			return
		case first == "--version":
			fmt.Printf("aster %s\n", Version)
			fmt.Printf("  commit: %s\n", Commit)
			fmt.Printf("  built:  %s\n", Date)
			return
		case first == "pick" || first == "p" || first == "-":
			path, err := ShowRecentPicker(nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			viewFile(path)
			return
		case first == "latest" || first == "l" || first == "+":
			path, err := GetNewestFile(nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Opening: %s\n", path)
			viewFile(path)
			return
		}

		// Subcommand mode: aster <type> [file|-|+]
		if ft, ok := fileTypes[first]; ok {
			runSubcommand(first, ft, os.Args[2:])
			return
		}

		// Hidden flag: -f <file>
		if first == "-f" && len(os.Args) >= 3 {
			filePath := expandPath(os.Args[2])
			viewTextFile(filePath, "", true)
			return
		}

		// Default: treat as file path
		filePath := expandPath(first)

		viewFile(filePath)
		return
	}

	// No args: try stdin
	if hasStdinData() {
		content, err := readStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		viewStdinContent(content, "")
		return
	}

	fmt.Fprintln(os.Stderr, "Error: No file provided.")
	fmt.Fprintln(os.Stderr, "Run 'aster help' for usage.")
	os.Exit(1)
}

// runSubcommand handles: aster <type> [file|-|+]
func runSubcommand(typeName string, ft fileType, args []string) {
	if len(args) == 0 {
		if hasStdinData() {
			content, err := readStdin()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			viewStdinContent(content, typeName)
			return
		}
		fmt.Fprintf(os.Stderr, "Usage: aster %s [file | - | +]\n\n", typeName)
		fmt.Fprintf(os.Stderr, "  aster %s <file>   View %s file\n", typeName, ft.name)
		fmt.Fprintf(os.Stderr, "  aster %s -        Pick from recent %s files\n", typeName, ft.name)
		fmt.Fprintf(os.Stderr, "  aster %s +        Open newest %s in cwd\n", typeName, ft.name)
		os.Exit(1)
	}

	// Handle -f as hidden flag within subcommand
	follow := false
	target := args[0]
	if target == "-f" && len(args) >= 2 {
		follow = true
		target = args[1]
	}

	filePath := resolveShortcut(target, ft.extensions)
	if filePath == "" {
		fmt.Fprintf(os.Stderr, "Usage: aster %s [file | - | +]\n", typeName)
		os.Exit(0)
	}

	if typeName == "img" {
		AddRecent(filePath)
		viewImage(filePath)
	} else {
		viewTextFile(filePath, typeName, follow)
	}
}
