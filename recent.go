package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxRecent = 5

// getRecentFile returns path to ~/.aster/recent
func getRecentFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".aster")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "recent"), nil
}

// AddRecent adds a file to recent history (deduped, max 5)
func AddRecent(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	recentFile, err := getRecentFile()
	if err != nil {
		return err
	}

	var lines []string
	if f, err := os.Open(recentFile); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		f.Close()
	}

	var newLines []string
	newLines = append(newLines, absPath)
	for _, line := range lines {
		if line != absPath && line != "" {
			newLines = append(newLines, line)
		}
	}

	if len(newLines) > maxRecent {
		newLines = newLines[:maxRecent]
	}

	f, err := os.Create(recentFile)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, line := range newLines {
		fmt.Fprintln(f, line)
	}
	return nil
}

// GetRecent returns recent files, optionally filtered by extensions
func GetRecent(exts []string) ([]string, error) {
	recentFile, err := getRecentFile()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(recentFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if len(exts) > 0 {
			ext := strings.ToLower(filepath.Ext(line))
			match := false
			for _, e := range exts {
				if ext == e {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// ShowRecentPicker displays recent files and returns selected path
func ShowRecentPicker(exts []string) (string, error) {
	recent, err := GetRecent(exts)
	if err != nil || len(recent) == 0 {
		return "", fmt.Errorf("no recent files")
	}

	fmt.Println("Recent files:")
	for i, path := range recent {
		display := path
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(rel, "..") {
				display = rel
			} else {
				display = filepath.Join(filepath.Base(filepath.Dir(path)), filepath.Base(path))
			}
		}
		fmt.Printf("  %d. %s\n", i+1, display)
	}

	fmt.Print("\n> ")
	var choice int
	fmt.Scanln(&choice)

	if choice < 1 || choice > len(recent) {
		return "", fmt.Errorf("invalid choice")
	}
	return recent[choice-1], nil
}

// GetNewestFile returns the most recently modified file in cwd, optionally filtered by extensions
func GetNewestFile(exts []string) (string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return "", err
	}

	type fileTime struct {
		path string
		time int64
	}

	var files []fileTime
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if len(exts) > 0 {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			match := false
			for _, e := range exts {
				if ext == e {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileTime{entry.Name(), info.ModTime().Unix()})
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no matching files in current directory")
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].time > files[j].time
	})

	return files[0].path, nil
}
