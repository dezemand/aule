package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GlobTool finds files matching a pattern
type GlobTool struct{}

type globInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

func (t *GlobTool) Name() string {
	return "glob"
}

func (t *GlobTool) Description() string {
	return `Find files matching a glob pattern.

Examples:
- "*.go" - all Go files in current directory
- "**/*.go" - all Go files recursively
- "src/**/*.ts" - all TypeScript files under src/
- "**/test_*.py" - all Python test files

Returns a list of matching file paths, sorted by modification time (newest first).`
}

func (t *GlobTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The glob pattern to match files (e.g., '**/*.go')",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Directory to search in. Defaults to working directory.",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobTool) Execute(ctx context.Context, workDir string, input json.RawMessage) (string, error) {
	var args globInput
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if args.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	// Resolve search path
	searchPath := workDir
	if args.Path != "" {
		if filepath.IsAbs(args.Path) {
			searchPath = args.Path
		} else {
			searchPath = filepath.Join(workDir, args.Path)
		}
	}
	searchPath = filepath.Clean(searchPath)

	// Check if path exists
	info, err := os.Stat(searchPath)
	if err != nil {
		return "", fmt.Errorf("path not found: %s", args.Path)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", args.Path)
	}

	// Find matching files
	var matches []fileInfo
	pattern := args.Pattern

	// Handle ** patterns (recursive)
	if strings.Contains(pattern, "**") {
		err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if info.IsDir() {
				// Skip hidden directories
				if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
					return filepath.SkipDir
				}
				return nil
			}

			// Get relative path
			relPath, _ := filepath.Rel(searchPath, path)

			// Match against pattern (simplified)
			if matchGlob(pattern, relPath) {
				matches = append(matches, fileInfo{
					path:    relPath,
					modTime: info.ModTime().Unix(),
				})
			}
			return nil
		})
	} else {
		// Simple glob (no **)
		fullPattern := filepath.Join(searchPath, pattern)
		globMatches, err := filepath.Glob(fullPattern)
		if err != nil {
			return "", fmt.Errorf("invalid pattern: %w", err)
		}

		for _, match := range globMatches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			relPath, _ := filepath.Rel(searchPath, match)
			matches = append(matches, fileInfo{
				path:    relPath,
				modTime: info.ModTime().Unix(),
			})
		}
	}

	if len(matches) == 0 {
		return "No files found matching pattern: " + args.Pattern, nil
	}

	// Sort by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime > matches[j].modTime
	})

	// Format output
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d files:\n", len(matches)))

	// Limit output
	maxFiles := 100
	for i, f := range matches {
		if i >= maxFiles {
			result.WriteString(fmt.Sprintf("\n... and %d more files", len(matches)-maxFiles))
			break
		}
		result.WriteString(f.path)
		result.WriteString("\n")
	}

	return result.String(), nil
}

type fileInfo struct {
	path    string
	modTime int64
}

// matchGlob is a simplified glob matcher that handles ** patterns
func matchGlob(pattern, path string) bool {
	// Convert ** pattern to regex-like matching
	parts := strings.Split(pattern, "**")

	if len(parts) == 1 {
		// No **, use simple match
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		return matched
	}

	// Handle **/ prefix
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		// Match suffix against filename or any path suffix
		if matched, _ := filepath.Match(suffix, filepath.Base(path)); matched {
			return true
		}
		// Also check full path
		pathParts := strings.Split(path, string(filepath.Separator))
		for i := range pathParts {
			subPath := filepath.Join(pathParts[i:]...)
			if matched, _ := filepath.Match(suffix, subPath); matched {
				return true
			}
		}
	}

	// Handle simple suffix patterns like *.go
	if strings.HasPrefix(pattern, "*") && !strings.Contains(pattern, "/") {
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		return matched
	}

	// Fallback: match the base name against the last part
	lastPart := parts[len(parts)-1]
	if lastPart != "" {
		matched, _ := filepath.Match(strings.TrimPrefix(lastPart, "/"), filepath.Base(path))
		return matched
	}

	return false
}
