package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditTool edits files using string replacement
type EditTool struct{}

type editInput struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

func (t *EditTool) Name() string {
	return "edit"
}

func (t *EditTool) Description() string {
	return `Edit a file by replacing text. Finds 'old_string' and replaces it with 'new_string'.

Guidelines:
- The old_string must match exactly (including whitespace and indentation)
- By default, only replaces the first occurrence
- Set replace_all=true to replace all occurrences
- Use this for targeted changes; for complete rewrites, use 'write' instead
- Include enough context in old_string to uniquely identify the target`
}

func (t *EditTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to edit",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "The exact text to find and replace",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The text to replace it with",
			},
			"replace_all": map[string]any{
				"type":        "boolean",
				"description": "If true, replace all occurrences. Default is false (first only).",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}

func (t *EditTool) Execute(ctx context.Context, workDir string, input json.RawMessage) (string, error) {
	var args editInput
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if args.OldString == "" {
		return "", fmt.Errorf("old_string is required")
	}

	// Resolve path
	path := args.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}
	path = filepath.Clean(path)

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", args.Path)
		}
		return "", fmt.Errorf("cannot read file: %w", err)
	}

	contentStr := string(content)

	// Check if old_string exists
	count := strings.Count(contentStr, args.OldString)
	if count == 0 {
		return "", fmt.Errorf("old_string not found in file")
	}

	// If not replace_all and multiple occurrences, error
	if !args.ReplaceAll && count > 1 {
		return "", fmt.Errorf("old_string found %d times - provide more context to uniquely identify the target, or set replace_all=true", count)
	}

	// Perform replacement
	var newContent string
	var replacements int

	if args.ReplaceAll {
		newContent = strings.ReplaceAll(contentStr, args.OldString, args.NewString)
		replacements = count
	} else {
		newContent = strings.Replace(contentStr, args.OldString, args.NewString, 1)
		replacements = 1
	}

	// Write file
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("cannot write file: %w", err)
	}

	if replacements == 1 {
		return fmt.Sprintf("Edited file: %s (1 replacement)", args.Path), nil
	}
	return fmt.Sprintf("Edited file: %s (%d replacements)", args.Path, replacements), nil
}
