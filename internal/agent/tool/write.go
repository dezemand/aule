package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteTool writes content to a file
type WriteTool struct{}

type writeInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *WriteTool) Name() string {
	return "write"
}

func (t *WriteTool) Description() string {
	return `Write content to a file. Creates the file if it doesn't exist, or overwrites if it does.
Parent directories will be created automatically.
Use this to create new files or completely replace file contents.
For making targeted changes to existing files, prefer the 'edit' tool instead.`
}

func (t *WriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The absolute or relative path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteTool) Execute(ctx context.Context, workDir string, input json.RawMessage) (string, error) {
	var args writeInput
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

	// Clean path
	path = filepath.Clean(path)

	// Create parent directories if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create directory: %w", err)
	}

	// Check if file exists (for reporting)
	_, err := os.Stat(path)
	isNew := os.IsNotExist(err)

	// Write file
	if err := os.WriteFile(path, []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("cannot write file: %w", err)
	}

	if isNew {
		return fmt.Sprintf("Created file: %s (%d bytes)", args.Path, len(args.Content)), nil
	}
	return fmt.Sprintf("Updated file: %s (%d bytes)", args.Path, len(args.Content)), nil
}
