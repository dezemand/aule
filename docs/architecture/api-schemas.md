# API Schemas

This document explains the API schema structure in the `/api` directory and how schemas flow from definition to runtime usage.

## Directory Structure

```
api/
├── auth.openapi.yaml      # REST API: OAuth/JWT authentication endpoints
├── agent.openapi.yaml     # REST API: Agent task execution API
└── ws/
    ├── messages.schema.yaml      # Core envelope + error types
    ├── subscription.schema.yaml  # Subscription lifecycle messages
    └── project.schema.yaml       # Project domain messages (39 types)
```

## Two API Styles

### REST APIs (`*.openapi.yaml`)

Used only for:
- **Authentication flows** (`auth.openapi.yaml`) - OAuth redirects, token exchange
- **Agent API** (`agent.openapi.yaml`) - Agent-to-backend communication

REST is used here because:
1. OAuth requires HTTP redirects
2. Agents are stateless workers that make one-off API calls
3. These don't benefit from real-time updates

Frontend uses Orval to generate typed clients:
```bash
cd frontend && bunx orval
```

### WebSocket Messages (`ws/*.schema.yaml`)

Used for all UI communication. Everything the frontend does goes through WebSocket:
- Project CRUD
- Member management
- Real-time subscriptions

Why WebSocket over REST for UI:
1. **Single connection** - No connection overhead per request
2. **Real-time updates** - Server pushes changes instantly via subscriptions
3. **Bidirectional** - Natural request/response and server-push in one protocol

## WebSocket Schema Files

### `messages.schema.yaml` - Core Types

Defines the fundamental message structure:

```yaml
definitions:
  Envelope:
    properties:
      type: string           # Message type (e.g., "projects.create.req")
      id: uuid               # Unique message ID
      reply_to: uuid         # Links response to request
      subscription_id: uuid  # For subscription-related messages
      seq: integer           # Ordering sequence number
      time: datetime         # ISO 8601 timestamp
      payload: object        # Message-specific data

  ErrorPayload:
    properties:
      code: string
      message: string
      detail: any
```

### `subscription.schema.yaml` - Subscription Lifecycle

Defines subscription management messages and query types:

```yaml
definitions:
  SubscribeRequest:
    properties:
      topic: string      # e.g., "projects.list", "projects.detail"
      query: object      # Topic-specific parameters
      initial: boolean   # Whether to send initial data

  ProjectsDetailQuery:
    properties:
      project_id: uuid

  ProjectsMembersQuery:
    properties:
      project_id: uuid
```

### `project.schema.yaml` - Domain Types

Defines all project-related types and messages (39 schemas):
- Domain models: `Project`, `ProjectMember`, `ProjectRepository`
- Nested objects: `ProjectPurpose`, `ProjectScope`, `ProjectGovernance`
- Request/response pairs for all operations

## Message Type Naming Convention

Messages follow a consistent naming pattern:

```
{domain}.{resource}.{action}.{direction}
```

Examples:
- `projects.list.req` - Request to list projects
- `projects.list` - Response with project list
- `projects.create.req` - Request to create a project
- `projects.created` - Response confirming creation
- `subscription.subscribe.req` - Request to start subscription
- `subscription.subscribe.ack` - Acknowledgment with subscription ID

## Schema to Code Flow

### Frontend (TypeScript)

1. YAML schemas are converted to Zod schemas by `frontend/scripts/generate-ws-schemas.ts`
2. Run: `cd frontend && bun run generate-ws-schemas`
3. Output: `frontend/src/model/ws/*.ts`

Generated code provides:
- Zod schemas for runtime validation
- TypeScript types inferred from schemas
- Message type constants

Example generated code:
```typescript
// From project.schema.yaml
export const projectSchema = z.object({
  id: z.string().uuid(),
  name: z.string(),
  status: z.enum(["active", "paused", "archived"]),
  // ...
});
export type Project = z.infer<typeof projectSchema>;
```

### Backend (Go)

The Go backend manually defines types in `internal/model/ws/`:
- `envelope.go` - Core `Envelope` type
- `projects.go` - Project message types
- `subscriptions.go` - Subscription message types

These must be kept in sync with the YAML schemas. The YAML schemas are the source of truth.

## Adding New Message Types

### 1. Define in YAML Schema

Add to the appropriate `api/ws/*.schema.yaml` file:

```yaml
definitions:
  MyNewRequest:
    type: object
    properties:
      foo: { type: string }
    required: [foo]

  MyNewResponse:
    type: object
    properties:
      result: { type: string }

messageTypes:
  my.new.req:
    payload: { $ref: "#/definitions/MyNewRequest" }
    direction: client-to-server
  my.new:
    payload: { $ref: "#/definitions/MyNewResponse" }
    direction: server-to-client
```

### 2. Regenerate Frontend Types

```bash
cd frontend && bun run generate-ws-schemas
```

### 3. Add Go Types

In `internal/model/ws/`:

```go
const (
    MsgTypeMyNewReq = "my.new.req"
    MsgTypeMyNew    = "my.new"
)

type MyNewRequest struct {
    Foo string `json:"foo"`
}

type MyNewResponse struct {
    Result string `json:"result"`
}
```

### 4. Implement Handler

In the appropriate service handler, use `WsToEvent` to transform incoming messages:

```go
wsproto.WsToEvent(bus, modelsws.MsgTypeMyNewReq, topicMyNew,
    func(payload modelsws.MyNewRequest, evt event.Event[eventsws.IncomingEvent]) myNewEvent {
        return myNewEvent{
            ClientID: evt.Payload().ClientID,
            UserID:   evt.Payload().UserID,
            Foo:      payload.Foo,
        }
    })
```
