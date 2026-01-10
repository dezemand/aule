package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BashTool executes shell commands
type BashTool struct {
	// Timeout for command execution
	Timeout time.Duration
}

type bashInput struct {
	Command     string `json:"command"`
	WorkDir     string `json:"workdir,omitempty"`
	Timeout     int    `json:"timeout,omitempty"` // in seconds
	Description string `json:"description,omitempty"`
}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Description() string {
	return `Execute a shell command in bash.

Guidelines:
- Commands run with a 2-minute timeout by default
- Use 'workdir' parameter to change directories instead of 'cd && command'
- For file operations, prefer dedicated tools (read, write, edit, glob, grep)
- Avoid interactive commands that require user input
- Be careful with destructive commands

Examples:
- {"command": "go build ./...", "description": "Build Go project"}
- {"command": "npm install", "workdir": "frontend", "description": "Install dependencies"}
- {"command": "git status", "description": "Check git status"}`
}

func (t *BashTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The bash command to execute",
			},
			"workdir": map[string]any{
				"type":        "string",
				"description": "Working directory for the command. Relative to project root.",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds. Default is 120.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Brief description of what this command does (5-10 words)",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BashTool) Execute(ctx context.Context, workDir string, input json.RawMessage) (string, error) {
	var args bashInput
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if args.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Determine working directory
	cmdWorkDir := workDir
	if args.WorkDir != "" {
		if strings.HasPrefix(args.WorkDir, "/") {
			cmdWorkDir = args.WorkDir
		} else {
			cmdWorkDir = workDir + "/" + args.WorkDir
		}
	}

	// Determine timeout
	timeout := t.Timeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	if args.Timeout > 0 {
		timeout = time.Duration(args.Timeout) * time.Second
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
	cmd.Dir = cmdWorkDir

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Build result
	var result strings.Builder

	if args.Description != "" {
		result.WriteString(fmt.Sprintf("$ %s\n", args.Description))
	}
	result.WriteString(fmt.Sprintf("$ %s\n", args.Command))

	if stdout.Len() > 0 {
		output := stdout.String()
		// Truncate very long output
		if len(output) > 50000 {
			output = output[:50000] + "\n... (output truncated)"
		}
		result.WriteString(output)
		if !strings.HasSuffix(output, "\n") {
			result.WriteString("\n")
		}
	}

	if stderr.Len() > 0 {
		stderrStr := stderr.String()
		if len(stderrStr) > 10000 {
			stderrStr = stderrStr[:10000] + "\n... (stderr truncated)"
		}
		result.WriteString("\nSTDERR:\n")
		result.WriteString(stderrStr)
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.WriteString(fmt.Sprintf("\n[Command timed out after %v]", timeout))
		} else {
			result.WriteString(fmt.Sprintf("\n[Exit code: %v]", err))
		}
	}

	return result.String(), nil
}
