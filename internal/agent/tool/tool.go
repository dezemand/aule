package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dezemandje/aule/internal/agent/llm"
)

// Tool is the interface that all tools must implement
type Tool interface {
	// Name returns the tool name (used in API calls)
	Name() string

	// Description returns a description of what the tool does
	Description() string

	// Parameters returns the JSON Schema for the tool's parameters
	Parameters() map[string]any

	// Execute runs the tool with the given input
	Execute(ctx context.Context, workDir string, input json.RawMessage) (string, error)
}

// Registry manages available tools
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools
func (r *Registry) All() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Names returns the names of all registered tools
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ToToolDefs converts registered tools to LLM tool definitions
func (r *Registry) ToToolDefs() []llm.ToolDef {
	defs := make([]llm.ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, llm.ToolDef{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}

// FilteredToolDefs returns tool definitions for only the specified tools
func (r *Registry) FilteredToolDefs(names []string) []llm.ToolDef {
	defs := make([]llm.ToolDef, 0, len(names))
	for _, name := range names {
		if t, ok := r.tools[name]; ok {
			defs = append(defs, llm.ToolDef{
				Type: "function",
				Function: llm.ToolFunction{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  t.Parameters(),
				},
			})
		}
	}
	return defs
}

// Execute runs a tool by name
func (r *Registry) Execute(ctx context.Context, workDir, name string, input json.RawMessage) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return t.Execute(ctx, workDir, input)
}

// DefaultRegistry creates a registry with all default tools
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(&ReadTool{})
	r.Register(&WriteTool{})
	r.Register(&EditTool{})
	r.Register(&GlobTool{})
	r.Register(&GrepTool{})
	r.Register(&BashTool{})
	return r
}
