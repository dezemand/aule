# OpenCode Agent & LLM Architecture

A deep dive into how OpenCode handles agents, connects to LLMs, and processes requests.

---

## Overview

OpenCode is built around a few core abstractions that work together to enable AI-powered coding assistance:

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Input                              │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Session & Prompt                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐    │
│  │  Agent   │  │ Provider │  │  Tools   │  │  Permission  │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      LLM Streaming                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                      │
│  │ AI SDK   │  │ Messages │  │ Processor│                      │
│  └──────────┘  └──────────┘  └──────────┘                      │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Tool Execution                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐    │
│  │   Bash   │  │   Edit   │  │   Read   │  │    Task      │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

**Key components:**

- **Agents** - Define behavior, permissions, and prompts for different use cases
- **Providers** - Connect to various LLM APIs (Anthropic, OpenAI, etc.)
- **Sessions** - Manage conversation state and message history
- **Tools** - Execute actions like file editing, shell commands, and searches
- **Permissions** - Control what tools and actions are allowed

---

## Agents

Agents are the primary abstraction for controlling AI behavior. Each agent has its own system prompt, permissions, and optional model override.

### Agent schema

```typescript
{
  name: string                    // Unique identifier
  description?: string            // When to use this agent
  mode: "primary" | "subagent" | "all"
  native?: boolean                // Built-in agent flag
  hidden?: boolean                // Hide from UI autocomplete
  temperature?: number            // LLM temperature
  topP?: number                   // LLM top-p sampling
  color?: string                  // Hex color for UI
  permission: Ruleset             // Tool access rules
  model?: {                       // Optional model override
    providerID: string
    modelID: string
  }
  prompt?: string                 // Custom system prompt
  options: Record<string, any>    // Additional LLM options
  steps?: number                  // Max agentic loop iterations
}
```

### Agent modes

| Mode       | Description                                 |
| ---------- | ------------------------------------------- |
| `primary`  | Main agents users interact with directly    |
| `subagent` | Spawned by primary agents via the Task tool |
| `all`      | Can be used as either primary or subagent   |

### Built-in agents

OpenCode ships with several native agents:

| Agent        | Mode             | Purpose                                                      |
| ------------ | ---------------- | ------------------------------------------------------------ |
| `build`      | primary          | General coding tasks, allows most tools including `question` |
| `plan`       | primary          | Planning mode, can only edit files in `.opencode/plan/`      |
| `general`    | subagent         | Multi-step parallel tasks, denies todo tools                 |
| `explore`    | subagent         | Fast codebase exploration (read-only tools)                  |
| `compaction` | primary (hidden) | Context summarization when token limit approached            |
| `title`      | primary (hidden) | Generate session titles                                      |
| `summary`    | primary (hidden) | Generate session summaries                                   |

### Configure custom agents

You can define custom agents in two ways:

**1. Via `opencode.json`:**

```json
{
  "agent": {
    "docs": {
      "model": "anthropic/claude-sonnet-4-20250514",
      "description": "Technical documentation writer",
      "mode": "subagent",
      "temperature": 0.7,
      "prompt": "You are a technical documentation expert...",
      "permission": {
        "bash": "deny",
        "edit": "allow"
      }
    }
  }
}
```

**2. Via markdown files in `.opencode/agent/`:**

Create `.opencode/agent/docs.md`:

```markdown
---
description: Technical documentation writer
mode: subagent
permission:
  bash: deny
  edit: allow
---

You are a technical documentation expert who writes clear,
concise documentation with examples.
```

### Agent loading

Agents are loaded at startup via `Agent.state()`:

1. Create default native agents with hardcoded permissions
2. Merge user-defined agents from config
3. Apply global permission overrides
4. Load markdown agent files from `.opencode/agent/`

---

## LLM Providers

OpenCode uses the Vercel AI SDK and supports 25+ LLM providers out of the box.

### Provider schema

```typescript
{
  id: string              // Provider identifier
  name: string            // Display name
  source: "env" | "config" | "custom" | "api"
  env: string[]           // Environment variable names for API key
  key?: string            // Resolved API key
  options: Record<string, any>
  models: Record<string, Model>
}
```

### Model schema

```typescript
{
  id: string
  providerID: string
  name: string
  api: {
    id: string            // Model ID for API calls
    url: string           // API endpoint
    npm: string           // SDK package name
  }
  capabilities: {
    temperature: boolean
    reasoning: boolean
    attachment: boolean
    toolcall: boolean
    input: { text, audio, image, video, pdf }
    output: { text, audio, image, video, pdf }
    interleaved: boolean
  }
  cost: {
    input: number         // Cost per 1M input tokens
    output: number        // Cost per 1M output tokens
    cache: { read, write }
  }
  limit: {
    context: number       // Max context window
    output: number        // Max output tokens
  }
  status: "alpha" | "beta" | "deprecated" | "active"
  variants?: Record<string, Record<string, any>>
}
```

### Supported providers

OpenCode bundles these provider SDKs directly:

| Provider              | SDK Package                   |
| --------------------- | ----------------------------- |
| Anthropic             | `@ai-sdk/anthropic`           |
| OpenAI                | `@ai-sdk/openai`              |
| Google Generative AI  | `@ai-sdk/google`              |
| Google Vertex AI      | `@ai-sdk/google-vertex`       |
| Amazon Bedrock        | `@ai-sdk/amazon-bedrock`      |
| Azure OpenAI          | `@ai-sdk/azure`               |
| OpenRouter            | `@openrouter/ai-sdk-provider` |
| xAI (Grok)            | `@ai-sdk/xai`                 |
| Mistral               | `@ai-sdk/mistral`             |
| Groq                  | `@ai-sdk/groq`                |
| DeepInfra             | `@ai-sdk/deepinfra`           |
| Cerebras              | `@ai-sdk/cerebras`            |
| Cohere                | `@ai-sdk/cohere`              |
| Cloudflare AI Gateway | `@ai-sdk/gateway`             |
| TogetherAI            | `@ai-sdk/togetherai`          |
| Perplexity            | `@ai-sdk/perplexity`          |
| Vercel                | `@ai-sdk/vercel`              |
| GitHub Copilot        | Custom OpenAI-compatible      |

Providers not bundled are dynamically installed at runtime using `BunProc.install()`.

### Provider initialization

The provider system initializes through `Provider.state()`:

```
1. Load model database from models.dev
2. Merge config providers from opencode.json
3. Check environment variables for API keys
4. Load stored auth credentials
5. Execute custom loaders for provider-specific init
6. Filter by enabled_providers/disabled_providers
```

### Authentication methods

| Method                | Description                                   |
| --------------------- | --------------------------------------------- |
| Environment Variables | `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.   |
| API Key Storage       | Stored in `~/.local/share/opencode/auth.json` |
| OAuth                 | For ChatGPT Plus/Pro, GitHub Copilot          |
| Plugin Auth           | Custom auth flows via plugins                 |

### Configure providers

```json
{
  "provider": {
    "anthropic": {
      "options": {
        "apiKey": "sk-...",
        "baseURL": "https://custom-endpoint.com",
        "timeout": 300000
      },
      "whitelist": ["claude-sonnet-4"],
      "blacklist": ["claude-opus"]
    }
  },
  "disabled_providers": ["google"],
  "enabled_providers": ["anthropic", "openai"],
  "model": "anthropic/claude-sonnet-4-20250514"
}
```

---

## The Agent Loop

The agent loop is the core execution engine that processes user messages and coordinates LLM responses.

### Entry points

| Function                  | Purpose                                |
| ------------------------- | -------------------------------------- |
| `SessionPrompt.prompt()`  | Send a user message and get a response |
| `SessionPrompt.command()` | Execute a slash command                |
| `SessionPrompt.shell()`   | Execute a shell command directly       |

### Main loop flow

The loop runs in `SessionPrompt.loop()`:

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. START                                                        │
│    - Check if session already running (abort if busy)           │
│    - Create AbortController for cancellation                    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. LOOP START                                                   │
│    - Set session status to "busy"                               │
│    - Load messages (filtered for compaction)                    │
│    - Find last user/assistant messages                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. CHECK EXIT CONDITIONS                                        │
│    - If assistant finished and user message older → break       │
│    - If abort signal triggered → break                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. HANDLE PENDING TASKS                                         │
│    - SUBTASK: Execute task tool, create synthetic user message  │
│    - COMPACTION: Run SessionCompaction.process()                │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. CHECK CONTEXT OVERFLOW                                       │
│    - If overflow detected → create compaction task, continue    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 6. NORMAL PROCESSING                                            │
│    - Get agent config                                           │
│    - Create assistant message placeholder                       │
│    - Resolve tools via resolveTools()                           │
│    - Create SessionProcessor                                    │
│    - Call processor.process() with LLM stream                   │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 7. PROCESS RESULT                                               │
│    - "stop" → break loop                                        │
│    - "compact" → create compaction, continue                    │
│    - "continue" → loop again (tool calls pending)               │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 8. END                                                          │
│    - Prune compaction                                           │
│    - Return last assistant message                              │
└─────────────────────────────────────────────────────────────────┘
```

### LLM streaming

The `LLM.stream()` function handles the actual API call:

```typescript
// Simplified flow
const stream = await streamText({
  model: wrapLanguageModel({
    model: language,
    middleware: [
      // Transform messages for provider compatibility
      { transformParams(args) { ... } },
      // Extract reasoning from <think> tags
      extractReasoningMiddleware({ tagName: "think" })
    ]
  }),
  messages: [...system, ...input.messages],
  tools,
  temperature,
  topP,
  maxOutputTokens,
  abortSignal
})
```

### Stream processing

The `SessionProcessor` handles stream events:

| Event          | Action                               |
| -------------- | ------------------------------------ |
| `start`        | Set status to busy                   |
| `reasoning-*`  | Track reasoning/thinking parts       |
| `tool-input-*` | Create tool part with pending state  |
| `tool-call`    | Set tool to running, check doom loop |
| `tool-result`  | Mark tool completed with output      |
| `tool-error`   | Mark tool failed, check if blocked   |
| `start-step`   | Take filesystem snapshot             |
| `finish-step`  | Calculate usage, create patch parts  |
| `text-*`       | Build text response parts            |
| `error`        | Handle errors, possibly retry        |
| `finish`       | Complete processing                  |

---

## Tools

Tools are the actions that agents can perform. Each tool has a schema, description, and execute function.

### Tool interface

```typescript
{
  id: string
  init: (ctx?) => Promise<{
    description: string
    parameters: ZodSchema
    execute(args, ctx: Tool.Context) => Promise<{
      title: string
      metadata: any
      output: string
      attachments?: FilePart[]
    }>
  }>
}
```

### Tool context

```typescript
{
  sessionID: string
  messageID: string
  agent: string
  abort: AbortSignal
  callID?: string
  extra?: Record<string, any>
  metadata(input): void      // Update tool state during execution
  ask(request): Promise<void> // Request permissions
}
```

### Built-in tools

| Tool         | Description                                                  |
| ------------ | ------------------------------------------------------------ |
| `bash`       | Execute shell commands                                       |
| `read`       | Read file contents                                           |
| `edit`       | Edit files with string replacements                          |
| `write`      | Write/create files                                           |
| `glob`       | Find files by pattern                                        |
| `grep`       | Search file contents with regex                              |
| `task`       | Spawn subagent for complex tasks                             |
| `webfetch`   | Fetch URL contents                                           |
| `websearch`  | Search the web (requires Exa or opencode provider)           |
| `codesearch` | Search code repositories (requires Exa or opencode provider) |
| `todoread`   | Read the todo list                                           |
| `todowrite`  | Update the todo list                                         |
| `question`   | Ask the user a question                                      |
| `skill`      | Load and execute skills                                      |
| `invalid`    | Handles malformed tool calls                                 |

### Tool registration

Tools are registered in the `ToolRegistry`:

```typescript
// Built-in tools
const tools = [
  BashTool,
  ReadTool,
  EditTool,
  WriteTool,
  GlobTool,
  GrepTool,
  TaskTool,
  WebFetchTool,
  TodoWriteTool,
  TodoReadTool,
  // ...
]

// Custom tools from .opencode/tool/*.ts
for await (const match of glob.scan({ cwd: configDir })) {
  const mod = await import(match)
  custom.push(fromPlugin(id, mod))
}

// Plugin tools
for (const plugin of plugins) {
  for (const [id, def] of Object.entries(plugin.tool ?? {})) {
    custom.push(fromPlugin(id, def))
  }
}
```

### Tool resolution

When processing a request, tools are resolved based on:

1. Provider capabilities (some providers don't support certain tools)
2. Agent permissions (deny/allow rules)
3. User-specified tool overrides

```typescript
async function resolveTools(input) {
  const disabled = PermissionNext.disabled(Object.keys(input.tools), input.agent.permission)

  for (const tool of Object.keys(input.tools)) {
    if (input.user.tools?.[tool] === false || disabled.has(tool)) {
      delete input.tools[tool]
    }
  }

  return input.tools
}
```

### Create custom tools

Create a file in `.opencode/tool/mytool.ts`:

```typescript
import { z } from "zod"

export default {
  description: "Does something useful",
  args: {
    input: z.string().describe("The input to process"),
  },
  async execute(args, ctx) {
    // Your tool logic here
    return `Processed: ${args.input}`
  },
}
```

---

## Subagents & Task Tool

The Task tool allows primary agents to spawn specialized subagents for complex tasks.

### How it works

1. Primary agent calls the `task` tool with a prompt and subagent type
2. Task tool creates a child session linked to the parent
3. Child session has restricted permissions (no todowrite, todoread, task)
4. Subagent executes and returns results to parent
5. Parent continues with the subagent's output

### Task tool parameters

```typescript
{
  description: string      // Short (3-5 words) task description
  prompt: string          // Detailed instructions for the subagent
  subagent_type: string   // Name of subagent to use (e.g., "explore")
  session_id?: string     // Continue existing session (optional)
  command?: string        // Command that triggered this task (optional)
}
```

### Child session permissions

Child sessions automatically deny:

- `todowrite` - Can't modify parent's todo list
- `todoread` - Can't read parent's todo list
- `task` - Can't spawn nested subagents

### Example flow

```
┌─────────────────────────────────────────────────────────────────┐
│ Primary Agent (build)                                           │
│                                                                 │
│ User: "Find all API endpoints in the codebase"                  │
│                                                                 │
│ Agent thinks: "This is an exploration task"                     │
│                                                                 │
│ → Calls task tool:                                              │
│   {                                                             │
│     description: "Find API endpoints",                          │
│     prompt: "Search for all API endpoint definitions...",       │
│     subagent_type: "explore"                                    │
│   }                                                             │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ Subagent (explore)                                              │
│                                                                 │
│ - Creates child session                                         │
│ - Runs with restricted permissions                              │
│ - Uses grep, glob, read tools                                   │
│ - Returns findings to parent                                    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ Primary Agent (build)                                           │
│                                                                 │
│ Receives subagent output and continues conversation             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Permission System

Permissions control what tools and actions are allowed per-agent.

### Permission rules

```typescript
{
  permission: string // Tool name or category
  pattern: string // Wildcard pattern for matching
  action: "allow" | "deny" | "ask"
}
```

### Special permissions

| Permission           | Purpose                                                      |
| -------------------- | ------------------------------------------------------------ |
| `doom_loop`          | Triggered when same tool called 3+ times with identical args |
| `external_directory` | Access to directories outside project                        |
| `question`           | Allows agent to ask user questions                           |
| `task`               | Controls which subagents can be invoked                      |

### Permission evaluation

```typescript
PermissionNext.evaluate(
  permission, // e.g., "bash", "edit", "read"
  pattern, // e.g., "*", "rm -rf *", "*.env"
  ...rulesets, // Agent ruleset + session overrides
)
// Returns: { action: "allow" | "deny" | "ask" }
```

### Default permissions

```typescript
const defaults = {
  "*": "allow",
  doom_loop: "ask",
  external_directory: {
    "*": "ask",
    [Truncate.DIR]: "allow",
  },
  question: "deny",
  read: {
    "*": "allow",
    "*.env": "deny",
    "*.env.*": "deny",
    "*.env.example": "allow",
  },
}
```

### Configure permissions

**Global permissions in `opencode.json`:**

```json
{
  "permission": {
    "bash": "ask",
    "edit": {
      "*.env": "deny",
      "src/**": "allow"
    }
  }
}
```

**Per-agent permissions:**

```json
{
  "agent": {
    "explore": {
      "permission": {
        "*": "deny",
        "read": "allow",
        "glob": "allow",
        "grep": "allow"
      }
    }
  }
}
```

---

## Sessions & Messages

Sessions manage conversation state and message history.

### Session schema

```typescript
{
  id: string               // Session identifier
  projectID: string        // Project this session belongs to
  directory: string        // Working directory
  parentID?: string        // Parent session (for subagents)
  title: string            // Session title
  version: string          // OpenCode version
  time: {
    created: number
    updated: number
    compacting?: number
    archived?: number
  }
  permission?: Ruleset     // Session-specific overrides
  revert?: RevertState     // Undo state
}
```

### Message types

**User message:**

```typescript
{
  id: string
  sessionID: string
  role: "user"
  time: { created: number }
  agent: string              // Which agent to use
  model: { providerID, modelID }
  system?: string            // Custom system prompt
  tools?: Record<string, boolean>
  variant?: string           // Reasoning effort variant
}
```

**Assistant message:**

```typescript
{
  id: string
  sessionID: string
  role: "assistant"
  parentID: string           // Links to user message
  time: { created, completed? }
  error?: ErrorType
  agent: string
  modelID: string
  providerID: string
  path: { cwd, root }
  cost: number               // Token cost
  tokens: { input, output, reasoning, cache }
  finish?: string            // Finish reason
}
```

### Message parts

Messages contain typed parts:

| Part Type                    | Description                                                 |
| ---------------------------- | ----------------------------------------------------------- |
| `text`                       | Plain text content                                          |
| `reasoning`                  | Model reasoning/thinking                                    |
| `tool`                       | Tool call with states (pending → running → completed/error) |
| `file`                       | File attachments                                            |
| `step-start` / `step-finish` | Mark processing steps                                       |
| `snapshot` / `patch`         | File change tracking                                        |
| `subtask`                    | Child task invocation                                       |
| `compaction`                 | Context compaction marker                                   |

### Tool states

```
pending → running → completed
                  → error
```

Each state has different data:

```typescript
// Pending
{ status: "pending", input, raw }

// Running
{ status: "running", input, title?, metadata?, time.start }

// Completed
{ status: "completed", input, output, title, metadata, time, attachments? }

// Error
{ status: "error", input, error, metadata?, time }
```

---

## System Prompts

OpenCode builds system prompts from multiple sources.

### Prompt structure

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Header (provider-specific)                                   │
│    - Anthropic spoof prompt for Claude models                   │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Provider/Agent Prompt                                        │
│    - Agent's custom prompt OR                                   │
│    - Provider-specific default (anthropic.txt, gemini.txt, etc) │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Environment Info                                             │
│    - Working directory                                          │
│    - Git status                                                 │
│    - Platform info                                              │
│    - Date                                                       │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Custom Instructions                                          │
│    - AGENTS.md / CLAUDE.md / CONTEXT.md                         │
│    - Global config AGENTS.md                                    │
│    - Config-specified instruction files/URLs                    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. User System Prompt (optional)                                │
│    - Custom prompt passed with message                          │
└─────────────────────────────────────────────────────────────────┘
```

### Instruction files

OpenCode looks for instruction files in this order:

**Local (project):**

1. `AGENTS.md`
2. `CLAUDE.md`
3. `CONTEXT.md` (deprecated)

**Global:**

1. `~/.config/opencode/AGENTS.md`
2. `~/.claude/CLAUDE.md`

---

## Event System

OpenCode uses an event bus for inter-component communication.

### Key events

| Event                         | Description               |
| ----------------------------- | ------------------------- |
| `Session.Event.Created`       | New session created       |
| `Session.Event.Updated`       | Session metadata changed  |
| `Session.Event.Deleted`       | Session removed           |
| `Session.Event.Error`         | Error occurred in session |
| `MessageV2.Event.Updated`     | Message changed           |
| `MessageV2.Event.PartUpdated` | Message part changed      |
| `Command.Event.Executed`      | Slash command executed    |

### Subscribe to events

```typescript
const unsub = Bus.subscribe(Session.Event.Created, (evt) => {
  console.log("New session:", evt.properties.session.id)
})

// Later...
unsub()
```

---

## Error Handling & Retries

### Retryable errors

The `SessionRetry` module handles transient errors:

- Rate limiting (429)
- Server errors (5xx)
- Network timeouts

```typescript
const retry = SessionRetry.retryable(error)
if (retry !== undefined) {
  const delay = SessionRetry.delay(attempt, error)
  await SessionRetry.sleep(delay, abort)
  // Continue loop...
}
```

### Non-retryable errors

- Permission denied (`PermissionNext.RejectedError`)
- User cancelled (`Question.RejectedError`)
- Invalid tool calls
- Model not found

---

## Plugin System

Plugins can extend OpenCode with custom tools, auth methods, and hooks.

### Plugin hooks

| Hook                                   | Purpose                    |
| -------------------------------------- | -------------------------- |
| `tool.execute.before`                  | Before tool execution      |
| `tool.execute.after`                   | After tool execution       |
| `chat.message`                         | When user message created  |
| `chat.params`                          | Modify LLM call parameters |
| `experimental.chat.system.transform`   | Transform system prompt    |
| `experimental.chat.messages.transform` | Transform message history  |
| `experimental.text.complete`           | Transform completed text   |

### Built-in plugins

- `opencode-copilot-auth` - GitHub Copilot authentication
- `opencode-anthropic-auth` - Anthropic OAuth
- `CodexAuthPlugin` - OpenAI Codex OAuth

---

## Configuration Reference

### Full config example

```json
{
  "model": "anthropic/claude-sonnet-4-20250514",
  "small_model": "anthropic/claude-haiku-4-5",
  "default_agent": "build",

  "provider": {
    "anthropic": {
      "options": {
        "timeout": 300000
      }
    }
  },

  "agent": {
    "custom": {
      "description": "Custom agent",
      "mode": "subagent",
      "prompt": "You are a specialized agent..."
    }
  },

  "permission": {
    "bash": "ask",
    "edit": {
      "*.env": "deny"
    }
  },

  "instructions": ["~/global-rules.md", "https://example.com/rules.txt"],

  "disabled_providers": ["google"],
  "enabled_providers": ["anthropic", "openai"],

  "experimental": {
    "continue_loop_on_deny": false,
    "batch_tool": false,
    "primary_tools": []
  }
}
```

---

## Quick Reference

### Start a session

```typescript
const session = await Session.create({ title: "My Session" })
const result = await SessionPrompt.prompt({
  sessionID: session.id,
  parts: [{ type: "text", text: "Hello!" }],
})
```

### Get an agent

```typescript
const agent = await Agent.get("build")
const agents = await Agent.list()
const defaultAgent = await Agent.defaultAgent()
```

### Get a provider/model

```typescript
const providers = await Provider.list()
const model = await Provider.getModel("anthropic", "claude-sonnet-4-20250514")
const language = await Provider.getLanguage(model)
```

### Register a custom tool

```typescript
await ToolRegistry.register({
  id: "mytool",
  init: async () => ({
    description: "My custom tool",
    parameters: z.object({ input: z.string() }),
    execute: async (args, ctx) => ({
      title: "Done",
      output: `Result: ${args.input}`,
      metadata: {},
    }),
  }),
})
```
