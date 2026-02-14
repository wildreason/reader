package main

import (
	"strings"
)

// Command represents a parsed user command
type Command struct {
	Action string // jump, next, prev, list, help, quit
	Arg    string // argument for jump command
}

// ParseCommand parses user input into a command
func ParseCommand(input string) *Command {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	parts := strings.Fields(input)
	action := strings.ToLower(parts[0])

	// Map single-letter keys to full commands
	actionMap := map[string]string{
		"j": "next",   // j = next
		"k": "prev",   // k = prev
		"l": "list",   // l = list
		"i": "jump",   // i = jump (input)
		"h": "help",   // h = help
		"q": "quit",   // q = quit
	}

	// Translate single-letter to full command
	if fullAction, exists := actionMap[action]; exists {
		action = fullAction
	}

	cmd := &Command{
		Action: action,
	}

	// Extract argument for jump command
	if action == "jump" && len(parts) > 1 {
		cmd.Arg = strings.Join(parts[1:], " ")
	}

	return cmd
}

// IsValid checks if command is valid
func (c *Command) IsValid() bool {
	validActions := map[string]bool{
		"jump": true,
		"next": true,
		"prev": true,
		"list": true,
		"help": true,
		"quit": true,
		"exit": true,
	}
	return validActions[c.Action]
}

// Navigator manages navigation state
type Navigator struct {
	index       *BlockIndex
	currentPos  int // Position in block list
	currentPage int // Current page within block (0-indexed)
	history     []int
	maxHistory  int
}

// NewNavigator creates a new navigator
func NewNavigator(index *BlockIndex) *Navigator {
	return &Navigator{
		index:       index,
		currentPos:  0,
		currentPage: 0,
		history:     []int{},
		maxHistory:  10,
	}
}

// ExecuteCommand processes a command and returns the result
func (nav *Navigator) ExecuteCommand(cmd *Command) (string, *Block, bool) {
	if cmd == nil {
		return "Invalid command. Type 'help' for available commands.", nil, false
	}

	if !cmd.IsValid() {
		return "Unknown command: " + cmd.Action + ". Type 'help' for available commands.", nil, false
	}

	switch cmd.Action {
	case "jump":
		return nav.handleJump(cmd.Arg)
	case "next":
		return nav.handleNext()
	case "prev":
		return nav.handlePrev()
	case "list":
		allNames := nav.index.GetAllBlockNames()
		return FormatBlockList(allNames), nil, false
	case "help":
		return FormatHelp(), nil, false
	case "quit", "exit":
		return "Goodbye!", nil, true // true indicates exit
	default:
		return "Unknown command.", nil, false
	}
}

// handleJump processes a jump command
func (nav *Navigator) handleJump(query string) (string, *Block, bool) {
	if query == "" {
		return "Usage: jump <block-name> (jump to a named block)", nil, false
	}

	block := nav.index.FindBlock(query)
	if block == nil {
		allNames := nav.index.GetAllBlockNames()
		msg := FormatNotFound(query, allNames)
		return msg, nil, false
	}

	// Update position
	lowerName := strings.ToLower(block.Name)
	if idx, ok := nav.index.nameIndex[lowerName]; ok {
		nav.saveHistory(nav.currentPos)
		nav.currentPos = idx
		nav.currentPage = 0 // Reset to first page of new block
	}

	return "", block, false
}

// handleNext jumps to the next block
func (nav *Navigator) handleNext() (string, *Block, bool) {
	if nav.currentPos+1 >= len(nav.index.blocks) {
		return "Already at the last block.", nil, false
	}

	nav.saveHistory(nav.currentPos)
	nav.currentPos++
	nav.currentPage = 0 // Reset to first page of new block
	block := nav.index.GetBlockByPosition(nav.currentPos)

	return "", block, false
}

// handlePrev jumps to the previous block
func (nav *Navigator) handlePrev() (string, *Block, bool) {
	if nav.currentPos <= 0 {
		return "Already at the first block.", nil, false
	}

	nav.saveHistory(nav.currentPos)
	nav.currentPos--
	nav.currentPage = 0 // Reset to first page of new block
	block := nav.index.GetBlockByPosition(nav.currentPos)

	return "", block, false
}

// saveHistory saves current position to history
func (nav *Navigator) saveHistory(pos int) {
	nav.history = append(nav.history, pos)
	if len(nav.history) > nav.maxHistory {
		nav.history = nav.history[1:]
	}
}

// GetCurrentBlock returns the current block
func (nav *Navigator) GetCurrentBlock() *Block {
	return nav.index.GetBlockByPosition(nav.currentPos)
}

// GetCurrentPosition returns current position in document
func (nav *Navigator) GetCurrentPosition() int {
	return nav.currentPos
}

// GetTotalBlocks returns total number of blocks
func (nav *Navigator) GetTotalBlocks() int {
	return len(nav.index.blocks)
}

// GetCurrentPage returns the current page (0-indexed)
func (nav *Navigator) GetCurrentPage() int {
	return nav.currentPage
}

// NextPage moves to the next page within current block
// Returns true if page changed, false if already at last page
func (nav *Navigator) NextPage() bool {
	block := nav.GetCurrentBlock()
	if block == nil {
		return false
	}

	if nav.currentPage+1 >= block.TotalPages {
		return false // Already at last page
	}

	nav.currentPage++
	return true
}

// PrevPage moves to the previous page within current block
// Returns true if page changed, false if already at first page
func (nav *Navigator) PrevPage() bool {
	if nav.currentPage <= 0 {
		return false // Already at first page
	}

	nav.currentPage--
	return true
}
