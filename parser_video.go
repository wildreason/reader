package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VideoParser implements FileParser for video files
type VideoParser struct{}

// Detect checks if file is a video
func (p *VideoParser) Detect(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, e := range videoExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// Parse is not used for video (binary content); returns nil
func (p *VideoParser) Parse(content string) []Block {
	return nil
}

// ParseFile reads a video file and returns a single Block with VideoData
// static=true inlines as base64 data URI (if <10MB); static=false stores file path for server mode
func (p *VideoParser) ParseFile(filePath string, static bool) ([]Block, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not resolve path: %w", err)
	}

	title := filepath.Base(filePath)
	ext := filepath.Ext(filePath)
	mime := videoMIME(ext)

	var src string
	inline := false

	if static {
		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("could not stat file: %w", err)
		}

		const maxInlineSize = 10 * 1024 * 1024 // 10MB
		if info.Size() >= maxInlineSize {
			fmt.Fprintf(os.Stderr, "Warning: %s is %dMB, too large to inline as base64. Referencing file path.\n",
				title, info.Size()/(1024*1024))
			src = absPath
			inline = false
		} else {
			data, err := os.ReadFile(absPath)
			if err != nil {
				return nil, fmt.Errorf("could not read file: %w", err)
			}
			b64 := base64.StdEncoding.EncodeToString(data)
			src = fmt.Sprintf("data:%s;base64,%s", mime, b64)
			inline = true
		}
	} else {
		src = absPath
	}

	block := Block{
		Name:        title,
		Content:     src,
		Pages:       []string{src},
		TotalPages:  1,
		ContentType: BlockContentVideo,
		Data: &VideoData{
			Src:    src,
			MIME:   mime,
			Inline: inline,
		},
	}

	return []Block{block}, nil
}
