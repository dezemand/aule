# Aule Agent Implementation

This document describes the agent implementation for Aule's autonomous task execution system.

## Overview

The agent is a standalone binary that executes tasks autonomously using LLM-powered reasoning and tool execution. It communicates with the backend via REST API to receive task assignments and report progress.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Agent Binary                             │
│                       (cmd/agent/main.go)                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │   Runner     │  │ LLM Provider │  │    Tool Registry     │   │
│  │  (loop)      │  │  (OpenAI)    │  │ (read,write,edit...) │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
│           │                │                    │                │
│           └────────────────┴────────────────────┘                │
│                            │                                     │
│                    ┌───────▼───────┐                             │
│                    │ Backend Client │                            │
│                    │   (REST API)   │                            │
│                    └───────┬───────┘                             │
└────────────────────────────┼────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Backend Server                             │
│                    (/agent/v1/tasks/...)                         │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### 1. LLM Provider (`internal/agent/llm/`)

OpenAI-compatible LLM client that supports:
- Chat completions with tool calling
- Configurable model, temperature, max tokens
- Token usage tracking

**Files:**
- `types.go` - Message, ContentBlock, ToolDef types
- `provider.go` - Provider interface
- `openai.go` - OpenAI API implementation

**Usage:**
```go
provider := llm.NewOpenAIProvider(llm.OpenAIConfig{
    APIKey:  os.Getenv("OPENAI_API_KEY"),
    BaseURL: "https://api.openai.com/v1", // or compatible endpoint
    Model:   "gpt-4o",
})

resp, err := provider.Complete(ctx, &llm.CompletionRequest{
    Messages: messages,
    Tools:    toolDefs,
})
```

### 2. Tool Framework (`internal/agent/tool/`)

Extensible tool system with built-in tools:

| Tool | Description |
|------|-------------|
| `read` | Read file contents with line numbers |
| `write` | Create or overwrite files |
| `edit` | Edit files with string replacement |
| `glob` | Find files by pattern |
| `grep` | Search file contents with regex |
| `bash` | Execute shell commands |

**Files:**
- `tool.go` - Tool interface and Registry
- `read.go`, `write.go`, `edit.go`, `glob.go`, `grep.go`, `bash.go` - Implementations

**Adding a new tool:**
```go
type MyTool struct{}

func (t *MyTool) Name() string { return "mytool" }
func (t *MyTool) Description() string { return "Does something" }
func (t *MyTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{...},
        "required": []string{...},
    }
}
func (t *MyTool) Execute(ctx context.Context, workDir string, input json.RawMessage) (string, error) {
    // Implementation
}

// Register it
registry.Register(&MyTool{})
```

### 3. Agent Runner (`internal/agent/runner/`)

The core agent loop that:
1. Sends messages to LLM
2. Processes responses (text + tool calls)
3. Executes tools and feeds results back
4. Repeats until task complete or max iterations

**Key features:**
- Configurable max iterations (default: 50)
- Progress callbacks for real-time reporting
- Token usage tracking
- Error handling and recovery

### 4. Backend Client (`internal/agent/client/`)

HTTP client for the agent API:
- `GetTask` - Fetch task details
- `StartTask` - Mark task as running
- `UpdateTask` - Send progress updates
- `CompleteTask` - Mark task complete
- `FailTask` - Mark task failed

### 5. Backend API (`internal/service/agentapi/`)

REST endpoints for agent communication:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/agent/v1/tasks/:task_id` | Get task details |
| POST | `/agent/v1/tasks/:task_id/start` | Start task execution |
| POST | `/agent/v1/tasks/:task_id/update` | Send progress update |
| POST | `/agent/v1/tasks/:task_id/complete` | Complete task |
| POST | `/agent/v1/tasks/:task_id/fail` | Fail task |

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | Yes | API key for OpenAI or compatible provider |
| `OPENAI_BASE_URL` | No | Custom API endpoint (default: OpenAI) |
| `OPENAI_MODEL` | No | Model to use (default: `gpt-4o`) |
| `TASK_ID` | No* | Task ID to execute |
| `TASK_AUTH_TOKEN` | No* | JWT for agent authentication |
| `AGENT_ENDPOINT` | No* | Backend URL |
| `WORK_DIR` | No | Working directory (default: cwd) |
| `STANDALONE` | No | Set to `true` for standalone mode |
| `AGENT_PROMPT` | No | Custom prompt for standalone mode |

*Required when not running in standalone mode

## Running the Agent

### Standalone Mode (Testing)

For testing without the backend:

```bash
export OPENAI_API_KEY=your-key
export STANDALONE=true
export AGENT_PROMPT="List all Go files in this project"

make agent
# or
./bin/agent
```

### With Backend

1. Start the backend:
```bash
make db-up
make migrate
make run
```

2. Create a task (via API or UI)

3. Run the agent:
```bash
export OPENAI_API_KEY=your-key
export TASK_ID=<task-uuid>
export AGENT_ENDPOINT=http://localhost:9000

./bin/agent
```

## Database Schema

The agent system uses these tables (migration `000002_tasks_agents`):

- `aule.agent_types` - Agent type definitions
- `aule.tasks` - Task definitions with execution context
- `aule.agent_instances` - Running/completed agent instances
- `aule.agent_logs` - Execution logs (tool calls, text output)

## Agent Loop Flow

```
1. INITIALIZE
   - Load config from environment
   - Create LLM provider, tool registry
   - Create backend client (if not standalone)

2. FETCH TASK
   - GET /agent/v1/tasks/{task_id}
   - Receive: title, description, context, allowed tools

3. START TASK
   - POST /agent/v1/tasks/{task_id}/start
   - Create agent instance, mark task running

4. AGENT LOOP (max 50 iterations)
   ┌──────────────────────────────────────┐
   │ a. Build messages array              │
   │ b. Call LLM with tools               │
   │ c. Process response:                 │
   │    - Text: accumulate as result      │
   │    - Tool use: execute, add result   │
   │    - Stop: break loop                │
   │ d. Send progress update              │
   └──────────────────────────────────────┘

5. COMPLETE
   - POST /agent/v1/tasks/{task_id}/complete
   - Or /fail if error occurred
```

## LLM Proxy

The LLM Proxy is a separate service that manages LLM API credentials and provides a unified interface for agents to access LLM providers. This eliminates the need for agents to have direct access to API keys.

### Architecture

```
Agent Binary                        Backend
    │                                  │
    │ GET /agent/v1/tasks/{id}         │
    ├─────────────────────────────────►│
    │◄─────────────────────────────────┤
    │   Returns: task + LLMConfig      │
    │   (endpoint, provider, model)    │
    │                                  │
    │                              LLM Proxy
    │ POST /v1/complete                │
    │ Headers:                         │
    │   Authorization: Bearer <jwt>    │
    │   X-LLM-Provider: openai         │
    │   X-LLM-Model: gpt-4o            │
    │   Accept: text/event-stream      │
    ├─────────────────────────────────►│
    │◄─────────────────────────────────┤
    │   SSE stream or JSON response    │
```

### LLM Proxy Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LLMPROXY_HOST` | No | `0.0.0.0` | Host to bind to |
| `LLMPROXY_PORT` | No | `9001` | Port to listen on |
| `JWT_SECRET` | Yes | - | Shared secret with backend for JWT validation |
| `OPENAI_API_KEY` | Yes* | - | OpenAI API key |
| `OPENAI_DEFAULT_MODEL` | No | `gpt-4o` | Default model for OpenAI |

*Required if using OpenAI provider

### Backend Configuration (for Proxy)

These variables configure what LLM settings the backend provides to agents:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LLMPROXY_ENDPOINT` | No | - | LLM Proxy endpoint URL |
| `LLM_DEFAULT_PROVIDER` | No | `openai` | Default LLM provider |
| `LLM_DEFAULT_MODEL` | No | `gpt-4o` | Default model |

### Running with LLM Proxy

1. Start all services:
```bash
# Terminal 1: Database
make db-up

# Terminal 2: Backend
make migrate
make run

# Terminal 3: LLM Proxy
export JWT_SECRET=your-shared-secret
export OPENAI_API_KEY=sk-...
make llmproxy
```

2. Agent receives LLM config from backend and uses proxy automatically.

### Proxy API

#### POST /v1/complete

Request completion from an LLM provider.

**Headers:**
- `Authorization: Bearer <jwt>` - Agent JWT token (required)
- `X-LLM-Provider: openai` - Provider name (required)
- `X-LLM-Model: gpt-4o` - Model name (optional, uses provider default)
- `Accept: text/event-stream` - For streaming (optional)

**Request Body:**
```json
{
  "messages": [
    {"role": "system", "content": "You are a helpful assistant"},
    {"role": "user", "content": [{"type": "text", "text": "Hello"}]}
  ],
  "tools": [
    {
      "name": "read",
      "description": "Read a file",
      "input_schema": {"type": "object", "properties": {...}}
    }
  ],
  "max_tokens": 4096,
  "temperature": 0.7
}
```

**Response (non-streaming):**
```json
{
  "content": [
    {"type": "text", "text": "Hello! How can I help?"}
  ],
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 10,
    "output_tokens": 8
  }
}
```

**Response (streaming SSE):**
```
event: content_block_delta
data: {"index":0,"type":"text","text":"Hello"}

event: content_block_delta
data: {"index":0,"type":"text","text":"!"}

event: message_done
data: {"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":8}}
```

### Provider Implementation

The proxy supports multiple LLM providers through the `Provider` interface:

```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan StreamEvent, error)
}
```

Currently implemented:
- **OpenAI** (`internal/llmproxy/provider/openai.go`) - GPT-4o and other OpenAI models

To add a new provider:
1. Implement the `Provider` interface
2. Register in `internal/llmproxy/provider/registry.go`

## Future Improvements

- [x] LLM Proxy for centralized API key management
- [x] Streaming responses via SSE
- [ ] WebSocket for real-time updates
- [x] Agent authentication (JWT middleware)
- [ ] PostgreSQL repositories (currently in-memory)
- [ ] Agent type-specific prompts and tool sets
- [ ] Kubernetes deployment with job controller
- [ ] Rate limiting and cost controls
- [ ] Multi-model support (Anthropic, etc.)
