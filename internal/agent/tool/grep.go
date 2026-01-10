package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// GrepTool searches file contents using regex
type GrepTool struct{}

type grepInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
	Include string `json:"include,omitempty"`
}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Description() string {
	return `Search file contents using regular expressions.

Examples:
- pattern: "func.*Handler" - find function definitions containing "Handler"
- pattern: "TODO|FIXME" - find TODO and FIXME comments
- pattern: "import.*react" - find React imports

Returns files with matching lines, sorted by modification time.`
}

func (t *GrepTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The regex pattern to search for",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Directory to search in. Defaults to working directory.",
			},
			"include": map[string]any{
				"type":        "string",
				"description": "File pattern to include (e.g., '*.go', '*.{ts,tsx}')",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(ctx context.Context, workDir string, input json.RawMessage) (string, error) {
	var args grepInput
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if args.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	// Compile regex
	re, err := regexp.Compile(args.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
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

	// Parse include patterns
	var includePatterns []string
	if args.Include != "" {
		// Handle {ts,tsx} style patterns
		if strings.Contains(args.Include, "{") {
			// Simple expansion of {a,b} patterns
			includePatterns = expandBraces(args.Include)
		} else {
			includePatterns = []string{args.Include}
		}
	}

	// Find and search files
	type match struct {
		file    string
		line    int
		content string
		modTime int64
	}

	var matches []match
	maxMatches := 200
	matchCount := 0

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Skip hidden directories and common non-code directories
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check include pattern
		if len(includePatterns) > 0 {
			matched := false
			for _, pattern := range includePatterns {
				if m, _ := filepath.Match(pattern, info.Name()); m {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		// Skip binary files (simple check)
		if isBinaryFile(info.Name()) {
			return nil
		}

		// Search file
		relPath, _ := filepath.Rel(searchPath, path)
		fileMatches := searchFile(path, re, maxMatches-matchCount)

		for _, fm := range fileMatches {
			matches = append(matches, match{
				file:    relPath,
				line:    fm.line,
				content: fm.content,
				modTime: info.ModTime().Unix(),
			})
			matchCount++
			if matchCount >= maxMatches {
				return filepath.SkipAll
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(matches) == 0 {
		return "No matches found for pattern: " + args.Pattern, nil
	}

	// Group by file and sort by mod time
	fileMatches := make(map[string][]match)
	fileModTimes := make(map[string]int64)

	for _, m := range matches {
		fileMatches[m.file] = append(fileMatches[m.file], m)
		if m.modTime > fileModTimes[m.file] {
			fileModTimes[m.file] = m.modTime
		}
	}

	// Sort files by mod time
	files := make([]string, 0, len(fileMatches))
	for f := range fileMatches {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool {
		return fileModTimes[files[i]] > fileModTimes[files[j]]
	})

	// Format output
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d matches in %d files:\n\n", len(matches), len(files)))

	for _, file := range files {
		result.WriteString(fmt.Sprintf("%s:\n", file))
		for _, m := range fileMatches[file] {
			content := m.content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			result.WriteString(fmt.Sprintf("  %d: %s\n", m.line, strings.TrimSpace(content)))
		}
		result.WriteString("\n")
	}

	if matchCount >= maxMatches {
		result.WriteString(fmt.Sprintf("(Results truncated at %d matches)", maxMatches))
	}

	return result.String(), nil
}

type fileMatch struct {
	line    int
	content string
}

func searchFile(path string, re *regexp.Regexp, maxMatches int) []fileMatch {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var matches []fileMatch
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() && len(matches) < maxMatches {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, fileMatch{
				line:    lineNum,
				content: line,
			})
		}
	}

	return matches
}

func isBinaryFile(name string) bool {
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
		".pdf": true, ".doc": true, ".docx": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
		".class": true, ".pyc": true, ".o": true, ".a": true,
	}
	ext := strings.ToLower(filepath.Ext(name))
	return binaryExts[ext]
}

func expandBraces(pattern string) []string {
	// Simple brace expansion: "*.{ts,tsx}" -> ["*.ts", "*.tsx"]
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")

	if start == -1 || end == -1 || end < start {
		return []string{pattern}
	}

	prefix := pattern[:start]
	suffix := pattern[end+1:]
	options := strings.Split(pattern[start+1:end], ",")

	var result []string
	for _, opt := range options {
		result = append(result, prefix+strings.TrimSpace(opt)+suffix)
	}
	return result
}
