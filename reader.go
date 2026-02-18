package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// renderAllContent renders all blocks and all their pages into a single string
func renderAllContent(blocks []Block, termWidth int, borderStyle BorderStyle) string {
	var out strings.Builder
	for i := range blocks {
		block := &blocks[i]
		for page := 0; page < block.TotalPages; page++ {
			rendered := FormatBlockPage(block, page, termWidth, borderStyle)
			out.WriteString(rendered)
		}
	}
	return out.String()
}

// runReaderMode runs the static reader TUI (non-follow mode)
func runReaderMode(blocks []Block, sourceName string, termWidth int, style string, borderStyle BorderStyle) {
	if len(blocks) == 0 {
		fmt.Println("Error: No blocks found in file.")
		return
	}

	// Initialize tview application and text view
	app := tview.NewApplication()

	text := tview.NewTextView().
		SetWrap(false).
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)

	// Render all content at once
	renderAll := func() {
		if showLineNumbers {
			SetLineNumbers(true, computeGutterWidth(blocks))
		} else {
			SetLineNumbers(false, 0)
		}
		content := renderAllContent(blocks, termWidth, borderStyle)
		text.Clear()
		fmt.Fprint(text, tview.TranslateANSI(content))
		text.ScrollToBeginning()
	}

	renderAll()

	// Key handling: j/k scroll, q quits
	text.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'j', 'J': // Scroll down
				row, col := text.GetScrollOffset()
				text.ScrollTo(row+3, col)
				return nil
			case 'k', 'K': // Scroll up
				row, col := text.GetScrollOffset()
				if row > 0 {
					newRow := row - 3
					if newRow < 0 {
						newRow = 0
					}
					text.ScrollTo(newRow, col)
				}
				return nil
			case 'd': // Half page down
				_, _, _, h := text.GetInnerRect()
				row, col := text.GetScrollOffset()
				text.ScrollTo(row+h/2, col)
				return nil
			case 'u': // Half page up
				_, _, _, h := text.GetInnerRect()
				row, col := text.GetScrollOffset()
				newRow := row - h/2
				if newRow < 0 {
					newRow = 0
				}
				text.ScrollTo(newRow, col)
				return nil
			case 'g': // Top of document
				text.ScrollToBeginning()
				return nil
			case 'G': // Bottom of document
				text.ScrollToEnd()
				return nil
			case 'q', 'Q':
				app.Stop()
				return nil
			}
		case tcell.KeyPgDn: // Page down
			_, _, _, h := text.GetInnerRect()
			row, col := text.GetScrollOffset()
			text.ScrollTo(row+h, col)
			return nil
		case tcell.KeyPgUp: // Page up
			_, _, _, h := text.GetInnerRect()
			row, col := text.GetScrollOffset()
			newRow := row - h
			if newRow < 0 {
				newRow = 0
			}
			text.ScrollTo(newRow, col)
			return nil
		case tcell.KeyCtrlC, tcell.KeyEscape:
			app.Stop()
			return nil
		}
		return ev
	})

	// Handle terminal resize
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		w, _ := screen.Size()
		if w != termWidth {
			termWidth = w
			renderAll()
		}
		return false
	})

	if err := app.SetRoot(text, true).Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
