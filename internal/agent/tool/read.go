package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadTool reads file contents
type ReadTool struct{}

type readInput struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"` // Line offset (0-based)
	Limit  int    `json:"limit,omitempty"`  // Max lines to read
}

func (t *ReadTool) Name() string {
	return "read"
}

func (t *ReadTool) Description() string {
	return `Read file contents from the filesystem. Returns the file content with line numbers.
Use this to examine source code, configuration files, or any text files.
For binary files or very large files, consider using other tools.`
}

func (t *ReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the file to read",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Line number to start reading from (0-based). Defaults to 0.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of lines to read. Defaults to 2000.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadTool) Execute(ctx context.Context, workDir string, input json.RawMessage) (string, error) {
	var args readInput
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Resolve path relative to workDir
	path := args.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}

	// Clean path to prevent traversal attacks
	path = filepath.Clean(path)

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", args.Path)
		}
		return "", fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", args.Path)
	}

	// Set defaults
	offset := args.Offset
	limit := args.Limit
	if limit <= 0 {
		limit = 2000
	}

	// Read file
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		if lineNum >= offset {
			if len(lines) >= limit {
				break
			}
			// Format: line_number<tab>content
			line := scanner.Text()
			// Truncate very long lines
			if len(line) > 2000 {
				line = line[:2000] + "... (truncated)"
			}
			lines = append(lines, fmt.Sprintf("%6d\t%s", lineNum+1, line))
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	if len(lines) == 0 {
		return "(empty file or offset beyond file length)", nil
	}

	result := strings.Join(lines, "\n")

	// Add truncation notice if needed
	if lineNum > offset+limit {
		result += fmt.Sprintf("\n\n(File truncated. Showing lines %d-%d of %d total lines)", offset+1, offset+len(lines), lineNum)
	}

	return result, nil
}
