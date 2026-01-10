# Automaker Agent & LLM Integration Documentation

This document provides comprehensive documentation on how Automaker handles its agents and connects to LLMs, including the abstractions used, connection methods, and the agent process lifecycle.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Provider Architecture (LLM Connections)](#provider-architecture-llm-connections)
3. [SDK Configuration Layer](#sdk-configuration-layer)
4. [Agent Services](#agent-services)
5. [Event System](#event-system)
6. [Security](#security)
7. [Data Storage](#data-storage)

---

## Architecture Overview

Automaker uses a layered architecture for AI agent operations:

```
┌─────────────────────────────────────────────────────────────────┐
│                         UI Layer                                │
│         (React hooks, WebSocket client, Electron IPC)           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Service Layer                              │
│    ┌──────────────────┐       ┌──────────────────────────┐      │
│    │   AgentService   │       │    AutoModeService       │      │
│    │  (Interactive    │       │   (Autonomous Feature    │      │
│    │     Chat)        │       │      Building)           │      │
│    └────────┬─────────┘       └───────────┬──────────────┘      │
└─────────────┼─────────────────────────────┼─────────────────────┘
              │                             │
              ▼                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Provider Layer                               │
│         ┌────────────────────────────────────┐                  │
│         │         ProviderFactory            │                  │
│         │   (Routes models to providers)     │                  │
│         └────────────────┬───────────────────┘                  │
│                          │                                      │
│         ┌────────────────▼───────────────────┐                  │
│         │         BaseProvider               │                  │
│         │      (Abstract Interface)          │                  │
│         └────────────────┬───────────────────┘                  │
│                          │                                      │
│         ┌────────────────▼───────────────────┐                  │
│         │        ClaudeProvider              │                  │
│         │ (@anthropic-ai/claude-agent-sdk)   │                  │
│         └────────────────────────────────────┘                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Configuration Layer                           │
│    ┌──────────────────────────────────────────────────────┐     │
│    │                  sdk-options.ts                      │     │
│    │  (Centralized SDK configuration & security validation)│    │
│    └──────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Event Layer                                │
│    ┌──────────────────────────────────────────────────────┐     │
│    │                  EventEmitter                        │     │
│    │         (WebSocket streaming to clients)             │     │
│    └──────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

### Two Agent Systems

Automaker has two distinct agent systems:

| System              | Purpose                                                | File Location                                   |
| ------------------- | ------------------------------------------------------ | ----------------------------------------------- |
| **AgentService**    | Interactive AI chat sessions with conversation history | `apps/server/src/services/agent-service.ts`     |
| **AutoModeService** | Autonomous feature implementation from a kanban board  | `apps/server/src/services/auto-mode-service.ts` |

Both systems use the same **Provider Architecture** to execute AI queries.

---

## Provider Architecture (LLM Connections)

The provider architecture enables Automaker to connect to different LLM providers through a unified interface.

### BaseProvider (Abstract Class)

**File:** `apps/server/src/providers/base-provider.ts`

The `BaseProvider` abstract class defines the contract that all provider implementations must follow:

```typescript
export abstract class BaseProvider {
  protected config: ProviderConfig;
  protected name: string;

  constructor(config: ProviderConfig = {}) {
    this.config = config;
    this.name = this.getName();
  }

  // Required abstract methods
  abstract getName(): string;
  abstract executeQuery(options: ExecuteOptions): AsyncGenerator<ProviderMessage>;
  abstract detectInstallation(): Promise<InstallationStatus>;
  abstract getAvailableModels(): ModelDefinition[];

  // Optional methods with default implementations
  validateConfig(): ValidationResult { ... }
  supportsFeature(feature: string): boolean { ... }
  getConfig(): ProviderConfig { ... }
  setConfig(config: Partial<ProviderConfig>): void { ... }
}
```

#### Key Methods

| Method                     | Description                                                 |
| -------------------------- | ----------------------------------------------------------- |
| `getName()`                | Returns provider identifier (e.g., "claude")                |
| `executeQuery(options)`    | Executes a query and streams responses via `AsyncGenerator` |
| `detectInstallation()`     | Checks if provider is installed and configured              |
| `getAvailableModels()`     | Returns array of available model definitions                |
| `supportsFeature(feature)` | Checks feature support (e.g., "vision", "tools")            |

### ClaudeProvider

**File:** `apps/server/src/providers/claude-provider.ts`

The `ClaudeProvider` implements `BaseProvider` using the official Anthropic Claude Agent SDK (`@anthropic-ai/claude-agent-sdk`).

```typescript
import { query, type Options } from '@anthropic-ai/claude-agent-sdk';

export class ClaudeProvider extends BaseProvider {
  getName(): string {
    return 'claude';
  }

  async *executeQuery(options: ExecuteOptions): AsyncGenerator<ProviderMessage> {
    const { prompt, model, cwd, systemPrompt, maxTurns, allowedTools,
            abortController, conversationHistory, sdkSessionId } = options;

    // Build SDK options
    const sdkOptions: Options = {
      model,
      systemPrompt,
      maxTurns,
      cwd,
      allowedTools: allowedTools || ['Read', 'Write', 'Edit', 'Glob', 'Grep', 'Bash', 'WebSearch', 'WebFetch'],
      permissionMode: 'acceptEdits',
      sandbox: { enabled: true, autoAllowBashIfSandboxed: true },
      abortController,
      // Resume existing session if available
      ...(sdkSessionId && conversationHistory?.length ? { resume: sdkSessionId } : {}),
    };

    // Execute via Claude Agent SDK
    const stream = query({ prompt: promptPayload, options: sdkOptions });

    // Stream messages directly
    for await (const msg of stream) {
      yield msg as ProviderMessage;
    }
  }

  async detectInstallation(): Promise<InstallationStatus> {
    const hasApiKey = !!process.env.ANTHROPIC_API_KEY;
    return {
      installed: true,
      method: 'sdk',
      hasApiKey,
      authenticated: hasApiKey,
    };
  }

  getAvailableModels(): ModelDefinition[] {
    return [
      { id: 'claude-opus-4-5-20251101', name: 'Claude Opus 4.5', tier: 'premium', default: true, ... },
      { id: 'claude-sonnet-4-20250514', name: 'Claude Sonnet 4', tier: 'standard', ... },
      { id: 'claude-3-5-sonnet-20241022', name: 'Claude 3.5 Sonnet', tier: 'standard', ... },
      { id: 'claude-haiku-4-5-20251001', name: 'Claude Haiku 4.5', tier: 'basic', ... },
    ];
  }

  supportsFeature(feature: string): boolean {
    return ['tools', 'text', 'vision', 'thinking'].includes(feature);
  }
}
```

#### Key Features

- **AsyncGenerator Streaming**: Returns an `AsyncGenerator<ProviderMessage>` for real-time response streaming
- **Session Resumption**: Supports resuming conversations via `sdkSessionId`
- **Multi-part Prompts**: Handles both text and image prompts
- **Sandbox Mode**: Enables safe bash execution with `autoAllowBashIfSandboxed`
- **Abort Support**: Integrates with `AbortController` for cancellation

### ProviderFactory

**File:** `apps/server/src/providers/provider-factory.ts`

The `ProviderFactory` routes model identifiers to the appropriate provider:

```typescript
export class ProviderFactory {
  static getProviderForModel(modelId: string): BaseProvider {
    const lowerModel = modelId.toLowerCase();

    // Claude models (claude-*, opus, sonnet, haiku)
    if (lowerModel.startsWith('claude-') || ['haiku', 'sonnet', 'opus'].includes(lowerModel)) {
      return new ClaudeProvider();
    }

    // Future providers:
    // if (lowerModel.startsWith("cursor-")) return new CursorProvider();
    // if (lowerModel.startsWith("opencode-")) return new OpenCodeProvider();

    // Default to Claude
    console.warn(`Unknown model prefix for "${modelId}", defaulting to Claude`);
    return new ClaudeProvider();
  }

  static getAllProviders(): BaseProvider[] { ... }
  static checkAllProviders(): Promise<Record<string, InstallationStatus>> { ... }
  static getProviderByName(name: string): BaseProvider | null { ... }
  static getAllAvailableModels(): ModelDefinition[] { ... }
}
```

### Type Definitions

**File:** `apps/server/src/providers/types.ts`

#### ExecuteOptions

Options passed to `provider.executeQuery()`:

```typescript
export interface ExecuteOptions {
  prompt: string | Array<{ type: string; text?: string; source?: object }>;
  model: string;
  cwd: string;
  systemPrompt?: string;
  maxTurns?: number;
  allowedTools?: string[];
  mcpServers?: Record<string, unknown>;
  abortController?: AbortController;
  conversationHistory?: ConversationMessage[];
  sdkSessionId?: string; // For resuming conversations
}
```

#### ProviderMessage

Messages yielded by providers (matches Claude SDK streaming format):

```typescript
export interface ProviderMessage {
  type: 'assistant' | 'user' | 'error' | 'result';
  subtype?: 'success' | 'error';
  session_id?: string;
  message?: {
    role: 'user' | 'assistant';
    content: ContentBlock[];
  };
  result?: string;
  error?: string;
  parent_tool_use_id?: string | null;
}
```

#### ContentBlock

Content blocks within messages:

```typescript
export interface ContentBlock {
  type: 'text' | 'tool_use' | 'thinking' | 'tool_result';
  text?: string;
  thinking?: string;
  name?: string; // Tool name for tool_use
  input?: unknown; // Tool input for tool_use
  tool_use_id?: string;
  content?: string; // Result for tool_result
}
```

#### ModelDefinition

```typescript
export interface ModelDefinition {
  id: string;
  name: string;
  modelString: string;
  provider: string;
  description: string;
  contextWindow?: number;
  maxOutputTokens?: number;
  supportsVision?: boolean;
  supportsTools?: boolean;
  tier?: 'basic' | 'standard' | 'premium';
  default?: boolean;
}
```

---

## SDK Configuration Layer

**File:** `apps/server/src/lib/sdk-options.ts`

The SDK options factory provides centralized configuration for all AI model invocations, including security validation.

### Tool Presets

```typescript
export const TOOL_PRESETS = {
  /** Read-only tools for analysis */
  readOnly: ['Read', 'Glob', 'Grep'],

  /** Tools for spec generation */
  specGeneration: ['Read', 'Glob', 'Grep'],

  /** Full tool access for implementation */
  fullAccess: ['Read', 'Write', 'Edit', 'Glob', 'Grep', 'Bash', 'WebSearch', 'WebFetch'],

  /** Tools for chat/interactive mode */
  chat: ['Read', 'Write', 'Edit', 'Glob', 'Grep', 'Bash', 'WebSearch', 'WebFetch'],
};
```

### Max Turns Presets

```typescript
export const MAX_TURNS = {
  quick: 50, // Quick operations
  standard: 100, // Standard operations
  extended: 250, // Long-running operations
  maximum: 1000, // Extensive exploration
};
```

### Model Resolution

Models are resolved with environment variable overrides:

```typescript
export function getModelForUseCase(
  useCase: 'spec' | 'features' | 'suggestions' | 'chat' | 'auto' | 'default',
  explicitModel?: string
): string {
  // Priority: explicit model > env variable > default
  if (explicitModel) return resolveModelString(explicitModel);

  const envVarMap = {
    spec: process.env.AUTOMAKER_MODEL_SPEC,
    features: process.env.AUTOMAKER_MODEL_FEATURES,
    suggestions: process.env.AUTOMAKER_MODEL_SUGGESTIONS,
    chat: process.env.AUTOMAKER_MODEL_CHAT,
    auto: process.env.AUTOMAKER_MODEL_AUTO,
    default: process.env.AUTOMAKER_MODEL_DEFAULT,
  };

  const envModel = envVarMap[useCase] || envVarMap.default;
  if (envModel) return resolveModelString(envModel);

  // Default models per use case
  const defaultModels = {
    spec: 'claude-haiku-4-5-20251001',
    features: 'claude-haiku-4-5-20251001',
    suggestions: 'claude-haiku-4-5-20251001',
    chat: 'claude-haiku-4-5-20251001',
    auto: 'claude-opus-4-5-20251101', // Premium model for autonomous work
    default: 'claude-opus-4-5-20251101',
  };

  return resolveModelString(defaultModels[useCase]);
}
```

### Factory Functions

Each factory function validates the working directory and returns configured SDK options:

| Function                           | Use Case                  | Tools        | Max Turns    |
| ---------------------------------- | ------------------------- | ------------ | ------------ |
| `createSpecGenerationOptions()`    | Spec generation           | readOnly     | 1000         |
| `createFeatureGenerationOptions()` | Feature JSON generation   | readOnly     | 50           |
| `createSuggestionsOptions()`       | Code suggestions          | readOnly     | 250          |
| `createChatOptions()`              | Interactive chat          | chat         | 100          |
| `createAutoModeOptions()`          | Autonomous implementation | fullAccess   | 1000         |
| `createCustomOptions()`            | Custom configuration      | configurable | configurable |

#### Example: createChatOptions

```typescript
export function createChatOptions(config: CreateSdkOptionsConfig): Options {
  // Security: Validate working directory
  validateWorkingDirectory(config.cwd);

  // Model priority: explicit > session > default
  const effectiveModel = config.model || config.sessionModel;

  return {
    permissionMode: 'acceptEdits',
    model: getModelForUseCase('chat', effectiveModel),
    maxTurns: MAX_TURNS.standard,
    cwd: config.cwd,
    allowedTools: [...TOOL_PRESETS.chat],
    sandbox: {
      enabled: true,
      autoAllowBashIfSandboxed: true,
    },
    ...(config.systemPrompt && { systemPrompt: config.systemPrompt }),
    ...(config.abortController && { abortController: config.abortController }),
  };
}
```

#### Example: createAutoModeOptions

```typescript
export function createAutoModeOptions(config: CreateSdkOptionsConfig): Options {
  validateWorkingDirectory(config.cwd);

  return {
    permissionMode: 'acceptEdits',
    model: getModelForUseCase('auto', config.model),
    maxTurns: MAX_TURNS.maximum,
    cwd: config.cwd,
    allowedTools: [...TOOL_PRESETS.fullAccess],
    sandbox: {
      enabled: true,
      autoAllowBashIfSandboxed: true,
    },
    ...(config.systemPrompt && { systemPrompt: config.systemPrompt }),
    ...(config.abortController && { abortController: config.abortController }),
  };
}
```

---

## Agent Services

### AgentService (Interactive Chat)

**File:** `apps/server/src/services/agent-service.ts`

The `AgentService` manages interactive chat sessions with conversation history and streaming responses.

#### Session Structure

```typescript
interface Session {
  messages: Message[];
  isRunning: boolean;
  abortController: AbortController | null;
  workingDirectory: string;
  model?: string;
  sdkSessionId?: string; // Claude SDK session ID for continuity
}

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  images?: Array<{ data: string; mimeType: string; filename: string }>;
  timestamp: string;
  isError?: boolean;
}
```

#### Agent Process Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│                    AgentService Lifecycle                       │
└─────────────────────────────────────────────────────────────────┘

1. INITIALIZATION (startConversation)
   ┌─────────────────────────────────────────────────────────────┐
   │ - Load session from disk (agent-sessions/{sessionId}.json)  │
   │ - Validate working directory against ALLOWED_ROOT_DIRECTORY │
   │ - Load SDK session ID for conversation continuity           │
   │ - Initialize session state in memory                        │
   └─────────────────────────────────────────────────────────────┘
                              │
                              ▼
2. EXECUTION (sendMessage)
   ┌─────────────────────────────────────────────────────────────┐
   │ - Add user message to session                               │
   │ - Set isRunning = true, create AbortController              │
   │ - Load context files (CLAUDE.md, CODE_QUALITY.md, etc.)     │
   │ - Build system prompt with context                          │
   │ - Get provider via ProviderFactory.getProviderForModel()    │
   │ - Call provider.executeQuery() -> AsyncGenerator            │
   │ - Stream responses via events.emit('agent:stream', {...})   │
   │ - Capture SDK session ID from first response                │
   │ - Save messages to disk after completion                    │
   └─────────────────────────────────────────────────────────────┘
                              │
                              ▼
3. STREAMING (for await loop)
   ┌─────────────────────────────────────────────────────────────┐
   │ For each message from provider:                             │
   │ - type='assistant': Extract text/tool_use blocks            │
   │   - Emit 'stream' events with partial content               │
   │   - Emit 'tool_use' events for tool invocations             │
   │ - type='result': Emit 'complete' event                      │
   │ - type='error': Handle and emit error                       │
   └─────────────────────────────────────────────────────────────┘
                              │
                              ▼
4. TERMINATION (stopExecution or completion)
   ┌─────────────────────────────────────────────────────────────┐
   │ - Call abortController.abort() if stopping                  │
   │ - Set isRunning = false                                     │
   │ - Clear abortController                                     │
   │ - Save final session state                                  │
   └─────────────────────────────────────────────────────────────┘
```

#### Key Methods

```typescript
export class AgentService {
  private sessions = new Map<string, Session>();

  // Start or resume a conversation
  async startConversation({ sessionId, workingDirectory }): Promise<{...}> {
    if (!this.sessions.has(sessionId)) {
      const messages = await this.loadSession(sessionId);
      validateWorkingDirectory(workingDirectory);
      this.sessions.set(sessionId, { messages, isRunning: false, ... });
    }
    return { success: true, messages, sessionId };
  }

  // Send a message and stream responses
  async sendMessage({ sessionId, message, workingDirectory, imagePaths, model }): Promise<{...}> {
    const session = this.sessions.get(sessionId);

    // Add user message
    session.messages.push(userMessage);
    session.isRunning = true;
    session.abortController = new AbortController();

    // Load context and build options
    const contextFilesPrompt = await loadContextFiles({ projectPath });
    const sdkOptions = createChatOptions({ cwd, model, systemPrompt });

    // Get provider and execute
    const provider = ProviderFactory.getProviderForModel(model);
    const stream = provider.executeQuery(options);

    // Process stream
    for await (const msg of stream) {
      // Capture SDK session ID
      if (msg.session_id && !session.sdkSessionId) {
        session.sdkSessionId = msg.session_id;
      }

      // Handle message types
      if (msg.type === 'assistant') {
        // Process text and tool_use blocks
        this.emitAgentEvent(sessionId, { type: 'stream', content, ... });
      } else if (msg.type === 'result') {
        this.emitAgentEvent(sessionId, { type: 'complete', ... });
      }
    }

    await this.saveSession(sessionId, session.messages);
    return { success: true, message: assistantMessage };
  }

  // Stop current execution
  async stopExecution(sessionId: string): Promise<{...}> {
    const session = this.sessions.get(sessionId);
    if (session?.abortController) {
      session.abortController.abort();
      session.isRunning = false;
    }
    return { success: true };
  }

  // Session management
  async clearSession(sessionId: string): Promise<{...}> { ... }
  async listSessions(includeArchived?: boolean): Promise<SessionMetadata[]> { ... }
  async createSession(name, projectPath?, workingDirectory?, model?): Promise<SessionMetadata> { ... }
  async deleteSession(sessionId: string): Promise<boolean> { ... }
}
```

#### Event Emission

```typescript
private emitAgentEvent(sessionId: string, data: Record<string, unknown>): void {
  this.events.emit('agent:stream', { sessionId, ...data });
}
```

Event types emitted:

- `{ type: 'message', message }` - New user message added
- `{ type: 'stream', messageId, content, isComplete }` - Streaming text content
- `{ type: 'tool_use', tool: { name, input } }` - Tool invocation
- `{ type: 'complete', messageId, content, toolUses }` - Execution complete
- `{ type: 'error', error, message }` - Error occurred

---

### AutoModeService (Autonomous Feature Building)

**File:** `apps/server/src/services/auto-mode-service.ts`

The `AutoModeService` manages autonomous feature implementation with planning phases, task tracking, and worktree support.

#### Core Structures

```typescript
interface RunningFeature {
  featureId: string;
  projectPath: string;
  worktreePath: string | null;
  branchName: string | null;
  abortController: AbortController;
  isAutoMode: boolean;
  startTime: number;
}

interface AutoLoopState {
  projectPath: string;
  maxConcurrency: number;
  abortController: AbortController;
  isRunning: boolean;
}

interface PlanSpec {
  status: 'pending' | 'generating' | 'generated' | 'approved' | 'rejected';
  content?: string;
  version: number;
  generatedAt?: string;
  approvedAt?: string;
  reviewedByUser: boolean;
  tasksCompleted?: number;
  tasksTotal?: number;
  currentTaskId?: string;
  tasks?: ParsedTask[];
}

interface ParsedTask {
  id: string; // e.g., "T001"
  description: string; // e.g., "Create user model"
  filePath?: string; // e.g., "src/models/user.ts"
  phase?: string; // e.g., "Phase 1: Foundation"
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
}
```

#### Planning Modes

The `AutoModeService` supports four planning modes:

| Mode   | Description                                 | Approval Required |
| ------ | ------------------------------------------- | ----------------- |
| `skip` | No planning phase, immediate implementation | No                |
| `lite` | Brief planning outline (5 items)            | Optional          |
| `spec` | Full specification with task breakdown      | Yes               |
| `full` | Comprehensive SDD with phases               | Yes               |

#### Planning Prompts

Planning prompts are defined for each mode:

```typescript
const PLANNING_PROMPTS = {
  lite: `## Planning Phase (Lite Mode)
    1. **Goal**: What are we accomplishing?
    2. **Approach**: How will we do it?
    3. **Files to Touch**: List files and changes
    4. **Tasks**: Numbered task list (3-7 items)
    5. **Risks**: Any gotchas to watch for
    After generating: "[PLAN_GENERATED]"`,

  spec: `## Specification Phase (Spec Mode)
    1. **Problem**: What problem are we solving?
    2. **Solution**: Brief approach
    3. **Acceptance Criteria**: GIVEN-WHEN-THEN format
    4. **Files to Modify**: Table of files
    5. **Implementation Tasks**: \`\`\`tasks block
    6. **Verification**: How to confirm feature works
    After generating: "[SPEC_GENERATED]" and wait for approval`,

  full: `## Full Specification Phase (Full SDD Mode)
    1. **Problem Statement**
    2. **User Story**
    3. **Acceptance Criteria** (happy path, edge cases, errors)
    4. **Technical Context**
    5. **Non-Goals**
    6. **Implementation Tasks** (phased)
    7. **Success Metrics**
    8. **Risks & Mitigations**
    After generating: "[SPEC_GENERATED]" and wait for approval`,
};
```

#### Feature Execution Lifecycle

````
┌─────────────────────────────────────────────────────────────────┐
│                 AutoModeService Lifecycle                       │
└─────────────────────────────────────────────────────────────────┘

1. INITIALIZATION (executeFeature)
   ┌─────────────────────────────────────────────────────────────┐
   │ - Add feature to runningFeatures Map                        │
   │ - Create AbortController                                    │
   │ - Validate project path against ALLOWED_ROOT_DIRECTORY      │
   │ - Check for existing context (resume if exists)             │
   │ - Load feature from .automaker/features/{id}/feature.json   │
   │ - Find/create worktree for isolated development             │
   │ - Emit 'auto_mode_feature_start' event                      │
   │ - Update feature status to 'in_progress'                    │
   └─────────────────────────────────────────────────────────────┘
                              │
                              ▼
2. PLANNING PHASE (if planningMode != 'skip')
   ┌─────────────────────────────────────────────────────────────┐
   │ - Build prompt with planning prefix                         │
   │ - Execute via provider.executeQuery()                       │
   │ - Detect [SPEC_GENERATED] marker in response                │
   │ - Parse tasks from ```tasks block                           │
   │ - Update planSpec with parsed tasks                         │
   │ - If requirePlanApproval:                                   │
   │   - Emit 'plan_approval_required' event                     │
   │   - Wait for user approval via Promise                      │
   │   - Handle revisions if rejected with feedback              │
   │   - Repeat until approved or cancelled                      │
   │ - Emit 'plan_approved' event                                │
   └─────────────────────────────────────────────────────────────┘
                              │
                              ▼
3. IMPLEMENTATION PHASE (runAgent)
   ┌─────────────────────────────────────────────────────────────┐
   │ - Build SDK options via createAutoModeOptions()             │
   │ - Get provider via ProviderFactory.getProviderForModel()    │
   │ - Build prompt with images using buildPromptWithImages()    │
   │ - Execute via provider.executeQuery()                       │
   │ - Stream processing loop:                                   │
   │   - Accumulate responseText                                 │
   │   - Debounced writes to agent-output.md                     │
   │   - Detect [TASK_START] and [TASK_COMPLETE] markers         │
   │   - Emit 'auto_mode_progress' events                        │
   │   - Emit 'auto_mode_tool' for tool invocations              │
   │   - Update task status in planSpec                          │
   └─────────────────────────────────────────────────────────────┘
                              │
                              ▼
4. COMPLETION
   ┌─────────────────────────────────────────────────────────────┐
   │ - Determine final status:                                   │
   │   - skipTests=true -> 'waiting_approval'                    │
   │   - skipTests=false -> 'verified'                           │
   │ - Update feature status                                     │
   │ - Emit 'auto_mode_feature_complete' event                   │
   │ - Remove from runningFeatures Map                           │
   └─────────────────────────────────────────────────────────────┘
````

#### Key Methods

```typescript
export class AutoModeService {
  private runningFeatures = new Map<string, RunningFeature>();
  private autoLoop: AutoLoopState | null = null;
  private pendingApprovals = new Map<string, PendingApproval>();

  // Start continuous auto mode loop
  async startAutoLoop(projectPath: string, maxConcurrency = 3): Promise<void> {
    this.autoLoopRunning = true;
    this.autoLoopAbortController = new AbortController();
    // Background loop picks pending features and executes them
    this.runAutoLoop();
  }

  // Stop the auto mode loop
  async stopAutoLoop(): Promise<number> {
    this.autoLoopRunning = false;
    this.autoLoopAbortController?.abort();
    return this.runningFeatures.size;
  }

  // Execute a single feature
  async executeFeature(
    projectPath: string,
    featureId: string,
    useWorktrees = false,
    isAutoMode = false,
    providedWorktreePath?: string,
    options?: { continuationPrompt?: string }
  ): Promise<void> {
    // Add to running features
    const abortController = new AbortController();
    this.runningFeatures.set(featureId, { featureId, projectPath, abortController, ... });

    // Load feature and setup worktree
    const feature = await this.loadFeature(projectPath, featureId);
    const workDir = worktreePath || projectPath;

    // Build prompt with planning phase
    const prompt = this.getPlanningPromptPrefix(feature) + this.buildFeaturePrompt(feature);

    // Run the agent
    await this.runAgent(workDir, featureId, prompt, abortController, projectPath, imagePaths, model, {
      planningMode: feature.planningMode,
      requirePlanApproval: feature.requirePlanApproval,
      systemPrompt: contextFilesPrompt,
    });

    // Update status and emit completion
    await this.updateFeatureStatus(projectPath, featureId, finalStatus);
    this.emitAutoModeEvent('auto_mode_feature_complete', { featureId, passes: true, ... });
  }

  // Stop a specific feature
  async stopFeature(featureId: string): Promise<boolean> {
    const running = this.runningFeatures.get(featureId);
    if (running) {
      this.cancelPlanApproval(featureId);
      running.abortController.abort();
      return true;
    }
    return false;
  }

  // Resume feature from saved context
  async resumeFeature(projectPath: string, featureId: string, useWorktrees = false): Promise<void> {
    const context = await this.loadAgentOutput(projectPath, featureId);
    if (context) {
      return this.executeFeatureWithContext(projectPath, featureId, context, useWorktrees);
    }
    return this.executeFeature(projectPath, featureId, useWorktrees, false);
  }

  // Follow up with additional instructions
  async followUpFeature(
    projectPath: string,
    featureId: string,
    prompt: string,
    imagePaths?: string[],
    useWorktrees = true
  ): Promise<void> {
    // Load previous context and build follow-up prompt
    const previousContext = await this.loadAgentOutput(projectPath, featureId);
    const fullPrompt = `## Follow-up on Feature Implementation\n${previousContext}\n## Follow-up Instructions\n${prompt}`;

    // Execute with planningMode='skip' (follow-ups don't need planning)
    await this.runAgent(workDir, featureId, fullPrompt, abortController, projectPath, imagePaths, model, {
      planningMode: 'skip',
      previousContent: previousContext,
    });
  }
}
```

#### The runAgent Method

The core execution method that handles the AI interaction:

```typescript
private async runAgent(
  workDir: string,
  featureId: string,
  prompt: string,
  abortController: AbortController,
  projectPath: string,
  imagePaths?: string[],
  model?: string,
  options?: {
    planningMode?: PlanningMode;
    requirePlanApproval?: boolean;
    previousContent?: string;
    systemPrompt?: string;
  }
): Promise<void> {
  // Build SDK options
  const sdkOptions = createAutoModeOptions({ cwd: workDir, model, abortController });

  // Get provider
  const provider = ProviderFactory.getProviderForModel(sdkOptions.model);

  // Build prompt with images
  const { content: promptContent } = await buildPromptWithImages(prompt, imagePaths, workDir);

  // Execute
  const stream = provider.executeQuery({
    prompt: promptContent,
    model: sdkOptions.model,
    maxTurns: sdkOptions.maxTurns,
    cwd: workDir,
    allowedTools: sdkOptions.allowedTools,
    abortController,
    systemPrompt: options?.systemPrompt,
  });

  let responseText = options?.previousContent ? `${previousContent}\n\n---\n\n## Follow-up Session\n\n` : '';

  // Stream processing loop
  for await (const msg of stream) {
    if (msg.type === 'assistant' && msg.message?.content) {
      for (const block of msg.message.content) {
        if (block.type === 'text') {
          responseText += block.text;

          // Check for [SPEC_GENERATED] marker
          if (planningModeRequiresApproval && responseText.includes('[SPEC_GENERATED]')) {
            // Parse tasks, wait for approval, handle revisions...
          }

          // Schedule incremental file write
          scheduleWrite();
        }
      }
    }
  }

  // Final write
  await writeToFile();
}
```

#### Plan Approval Workflow

When `requirePlanApproval` is true:

```typescript
// Wait for user to approve or reject the plan
const approvalPromise = this.waitForPlanApproval(featureId, projectPath);

// Emit event for UI to show approval dialog
this.emitAutoModeEvent('plan_approval_required', {
  featureId,
  projectPath,
  planContent,
  planningMode,
  planVersion,
});

// Wait for user response
const approvalResult = await approvalPromise;

if (approvalResult.approved) {
  // Continue with implementation
  if (approvalResult.editedPlan) {
    // Use edited version
  }
} else {
  if (approvalResult.feedback) {
    // Regenerate plan with feedback
    const revisionStream = provider.executeQuery({ prompt: revisionPrompt, ... });
    // Parse new plan and repeat approval loop
  } else {
    // User cancelled
    throw new Error('Plan cancelled by user');
  }
}
```

#### Task Progress Tracking

Tasks are tracked via markers in the agent output:

```
[TASK_START] T001: Create user model
[TASK_COMPLETE] T001: Created user.ts with User interface and validation
[TASK_START] T002: Add API endpoint
...
```

These markers are detected and used to update the UI in real-time.

---

## Event System

**File:** `apps/server/src/lib/events.ts`

The event system enables real-time streaming to WebSocket clients.

### EventEmitter Interface

```typescript
export interface EventEmitter {
  emit: (type: EventType, payload: unknown) => void;
  subscribe: (callback: EventCallback) => () => void;
}

export function createEventEmitter(): EventEmitter {
  const subscribers = new Set<EventCallback>();

  return {
    emit(type: EventType, payload: unknown) {
      for (const callback of subscribers) {
        try {
          callback(type, payload);
        } catch (error) {
          console.error('Error in event subscriber:', error);
        }
      }
    },

    subscribe(callback: EventCallback) {
      subscribers.add(callback);
      return () => subscribers.delete(callback);
    },
  };
}
```

### Event Types

#### Agent Events (from AgentService)

| Event          | Description                                     |
| -------------- | ----------------------------------------------- |
| `agent:stream` | All agent-related events with nested type field |

Nested types:

- `message` - New message added
- `stream` - Streaming content update
- `tool_use` - Tool invocation
- `complete` - Execution complete
- `error` - Error occurred

#### Auto Mode Events (from AutoModeService)

| Event             | Description                                 |
| ----------------- | ------------------------------------------- |
| `auto-mode:event` | All auto-mode events with nested type field |

Nested types:

- `auto_mode_started` - Auto loop started
- `auto_mode_stopped` - Auto loop stopped
- `auto_mode_idle` - No pending features
- `auto_mode_feature_start` - Feature execution started
- `auto_mode_progress` - Text content streaming
- `auto_mode_tool` - Tool invocation
- `auto_mode_feature_complete` - Feature execution complete
- `auto_mode_error` - Error occurred
- `planning_started` - Planning phase started
- `plan_approval_required` - Waiting for user approval
- `plan_approved` - Plan was approved
- `plan_revision_requested` - User requested changes
- `auto_mode_task_started` - Individual task started
- `auto_mode_task_complete` - Individual task complete

### WebSocket Server

**File:** `apps/server/src/index.ts`

Events are delivered via WebSocket:

```typescript
// WebSocket endpoint for events
app.get('/api/events', (ctx) => {
  if (ctx.request.header('upgrade') === 'websocket') {
    // Upgrade to WebSocket
    const ws = ctx.upgrade();

    // Subscribe to events
    const unsubscribe = events.subscribe((type, payload) => {
      ws.send(JSON.stringify({ type, payload }));
    });

    ws.onclose = () => unsubscribe();
  }
});
```

---

## Security

### Path Validation

**File:** `apps/server/src/lib/sdk-options.ts`

All agent operations validate working directories against `ALLOWED_ROOT_DIRECTORY`:

```typescript
export function validateWorkingDirectory(cwd: string): void {
  const resolvedCwd = path.resolve(cwd);

  if (!isPathAllowed(resolvedCwd)) {
    const allowedRoot = getAllowedRootDirectory();
    throw new PathNotAllowedError(
      `Working directory "${cwd}" (resolved: ${resolvedCwd}) is not allowed. ` +
        `Must be within ALLOWED_ROOT_DIRECTORY: ${allowedRoot}`
    );
  }
}
```

This validation is called by:

- All `create*Options()` factory functions
- `AgentService.startConversation()`
- `AgentService.createSession()`
- `AutoModeService.executeFeature()`
- `AutoModeService.followUpFeature()`

### Sandbox Mode

The Claude SDK is configured with sandbox mode for safe bash execution:

```typescript
const sdkOptions: Options = {
  permissionMode: 'acceptEdits',
  sandbox: {
    enabled: true,
    autoAllowBashIfSandboxed: true,
  },
  // ...
};
```

### API Key Management

**File:** `apps/server/src/routes/setup/routes/store-api-key.ts`

API keys are stored securely:

```typescript
// Store in environment variable
process.env.ANTHROPIC_API_KEY = apiKey;

// Persist to .env file
await persistApiKeyToEnv('ANTHROPIC_API_KEY', apiKey);
```

---

## Data Storage

### Directory Structure

```
.automaker/
├── features/
│   └── {featureId}/
│       ├── feature.json       # Feature definition
│       ├── agent-output.md    # Agent execution log
│       └── images/            # Context images
├── agent-sessions/
│   └── {sessionId}.json       # Chat session messages
├── sessions-metadata.json     # Session index
└── app_spec.txt               # Project specification
```

### Feature Storage

**File:** `apps/server/src/services/feature-loader.ts`

Features are stored in individual folders:

```typescript
// Feature path: .automaker/features/{id}/feature.json
const featureDir = getFeatureDir(projectPath, featureId);
const featurePath = path.join(featureDir, 'feature.json');

// Agent output: .automaker/features/{id}/agent-output.md
const outputPath = path.join(featureDir, 'agent-output.md');
```

### Session Storage

Chat sessions are stored as JSON files:

```typescript
// Session path: {dataDir}/agent-sessions/{sessionId}.json
const sessionFile = path.join(this.stateDir, `${sessionId}.json`);

// Contains array of Message objects
const messages: Message[] = JSON.parse(await fs.readFile(sessionFile, 'utf-8'));
```

### Session Metadata

Session index is maintained separately:

```typescript
// Metadata path: {dataDir}/sessions-metadata.json
interface SessionMetadata {
  id: string;
  name: string;
  projectPath?: string;
  workingDirectory: string;
  createdAt: string;
  updatedAt: string;
  archived?: boolean;
  tags?: string[];
  model?: string;
  sdkSessionId?: string;
}
```

---

## Summary

Automaker's agent system is built on a clean, extensible architecture:

1. **Provider Architecture**: Abstract `BaseProvider` interface with `ClaudeProvider` implementation using the official Claude Agent SDK. The `ProviderFactory` routes models to providers.

2. **SDK Configuration**: Centralized configuration via `sdk-options.ts` with presets for different use cases, security validation, and model resolution.

3. **Agent Services**:
   - `AgentService` for interactive chat with session persistence and conversation continuity
   - `AutoModeService` for autonomous feature building with planning phases, task tracking, and worktree support

4. **Event System**: Real-time streaming via `EventEmitter` to WebSocket clients for progress updates.

5. **Security**: Path validation against `ALLOWED_ROOT_DIRECTORY` and sandbox mode for safe execution.

This architecture makes it straightforward to:

- Add new LLM providers by implementing `BaseProvider`
- Create new agent use cases via factory functions
- Customize model selection via environment variables
- Track progress in real-time via the event system
