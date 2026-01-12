# Backend Models

This document explains the model/type organization in the Go backend.

## Directory Structure

```
internal/
├── domain/              # Core business entities (source of truth)
│   └── project.go       # Project, ProjectMember, ProjectStatus, etc.
│
├── model/
│   ├── ws/              # WebSocket message types
│   │   ├── envelope.go  # Envelope, ErrorPayload
│   │   ├── projects.go  # Project CRUD messages
│   │   └── subscriptions.go
│   │
│   ├── events/
│   │   ├── ws/          # WebSocket lifecycle events
│   │   │   └── events.go    # ConnectEvent, IncomingEvent, OutgoingEvent
│   │   └── projects/    # Domain events
│   │       └── events.go    # ProjectCreatedEvent, MemberAddedEvent
│   │
│   └── http/            # HTTP API request/response types
│       └── agent.go     # Agent API types
│
└── service/
    └── project/
        ├── service.go   # Business logic
        └── handler.go   # WebSocket handlers
```

## Model Categories

### 1. Domain Models (`internal/domain/`)

Core business entities. These are the source of truth for data structures.

```go
// domain/project.go
type Project struct {
    ID          ProjectID          `json:"id"`
    Key         string             `json:"key"`
    Name        string             `json:"name"`
    Status      ProjectStatus      `json:"status"`
    Purpose     *ProjectPurpose    `json:"purpose,omitempty"`
    Scope       *ProjectScope      `json:"scope,omitempty"`
    Governance  *ProjectGovernance `json:"governance,omitempty"`
    // ...
}

type ProjectMember struct {
    ID        uuid.UUID                 `json:"id"`
    ProjectID ProjectID                 `json:"project_id"`
    UserID    UserID                    `json:"user_id"`
    Role      ProjectMemberRole         `json:"role"`
    // ...
}
```

Domain models:
- Are used by repositories, services, and handlers
- Are serialized directly in WebSocket responses
- Define enums with typed constants (e.g., `ProjectStatus`)
- Include JSON tags for API serialization

### 2. WebSocket Message Types (`internal/model/ws/`)

Request/response types for WebSocket communication.

```go
// model/ws/projects.go

// Message type constants
const (
    MsgTypeProjectCreate  = "projects.create.req"
    MsgTypeProjectCreated = "projects.created"
    // ...
)

// Request type
type ProjectCreateRequest struct {
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}

func (p *ProjectCreateRequest) Type() string { return MsgTypeProjectCreate }

// Response type (wraps domain model)
type ProjectCreatedResponse struct {
    Project domain.Project `json:"project"`
}

func (p *ProjectCreatedResponse) Type() string { return MsgTypeProjectCreated }
```

Conventions:
- Request types end with `Request`
- Response types end with `Response`
- Each type implements `Type() string` method
- Message type constants follow `MsgType{Action}` pattern
- Responses often wrap domain models

### 3. Event Types (`internal/model/events/`)

Events are used for decoupled communication between components.

#### WebSocket Events (`model/events/ws/`)

Internal events for WebSocket lifecycle and message routing:

```go
// model/events/ws/events.go

// Topics
var (
    TopicConnect    = event.NewTopic[ConnectEvent]("ws.connect")
    TopicDisconnect = event.NewTopic[DisconnectEvent]("ws.disconnect")
    TopicIncoming   = event.NewTopic[IncomingEvent]("ws.message.incoming")
    TopicOutgoing   = event.NewTopic[OutgoingEvent]("ws.message.outgoing")
)

// IncomingEvent is published when a message is received
type IncomingEvent struct {
    WsEvent
    Message *modelsws.Envelope
}

// OutgoingEvent is published to send messages to clients
type OutgoingEvent struct {
    To             []OutgoingTo
    Type           string
    Payload        json.RawMessage
    SubscriptionID *uuid.UUID
}
```

#### Domain Events (`model/events/{domain}/`)

Business events published when data changes:

```go
// model/events/projects/events.go

// Topics
var (
    TopicProjectCreated = event.NewTopic[ProjectCreatedEvent]("projects.created")
    TopicProjectUpdated = event.NewTopic[ProjectUpdatedEvent]("projects.updated")
    TopicMemberAdded    = event.NewTopic[MemberAddedEvent]("projects.members.added")
    // ...
)

// Event payloads
type ProjectCreatedEvent struct {
    ProjectID domain.ProjectID
    CreatorID domain.UserID
    Project   domain.Project
}

type MemberAddedEvent struct {
    ProjectID    domain.ProjectID
    MemberUserID domain.UserID
    Role         domain.ProjectMemberRole
    AddedBy      domain.UserID
}
```

Domain events:
- Are published from service layer after state changes
- Trigger subscription notifications
- Enable decoupled reactions (logging, notifications, etc.)

### 4. HTTP Types (`internal/model/http/`)

Request/response types for REST APIs (auth, agent):

```go
// model/http/agent.go
type TaskGetResponse struct {
    Task   AgentTask `json:"task"`
    Config LLMConfig `json:"config"`
}
```

## Data Flow

### Incoming WebSocket Message

```
1. WebSocket receives JSON
   │
   ▼
2. Parse into modelsws.Envelope
   │
   ▼
3. Publish eventsws.IncomingEvent
   │
   ▼
4. WsToEvent transformer matches type
   │
   ▼
5. Decode payload into modelsws.ProjectCreateRequest
   │
   ▼
6. Transform to internal event (createProjectEvent)
   │
   ▼
7. Handler calls service.CreateProject()
   │
   ▼
8. Service returns domain.Project
   │
   ▼
9. Publish eventsws.OutgoingEvent with modelsws.ProjectCreatedResponse
```

### Domain Event to Subscription Notification

```
1. Service method completes (e.g., AddMember)
   │
   ▼
2. Publish domain event (eventsprojects.TopicMemberAdded)
   │
   ▼
3. SubscribeToBus bridge receives event
   │
   ▼
4. Filter finds matching subscriptions
   │
   ▼
5. Build modelsws.MembersListResponse
   │
   ▼
6. Publish eventsws.OutgoingEvent for each subscriber
```

## Adding New Domain Models

### 1. Define in Domain Package

```go
// internal/domain/widget.go
type WidgetID uuid.UUID

type Widget struct {
    ID        WidgetID  `json:"id"`
    Name      string    `json:"name"`
    ProjectID ProjectID `json:"project_id"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 2. Add WS Message Types

```go
// internal/model/ws/widgets.go
const (
    MsgTypeWidgetCreate  = "widgets.create.req"
    MsgTypeWidgetCreated = "widgets.created"
)

type WidgetCreateRequest struct {
    ProjectID string `json:"project_id"`
    Name      string `json:"name"`
}

type WidgetCreatedResponse struct {
    Widget domain.Widget `json:"widget"`
}
```

### 3. Add Domain Events

```go
// internal/model/events/widgets/events.go
var (
    TopicWidgetCreated = event.NewTopic[WidgetCreatedEvent]("widgets.created")
)

type WidgetCreatedEvent struct {
    WidgetID  domain.WidgetID
    ProjectID domain.ProjectID
    CreatorID domain.UserID
}
```

### 4. Update API Schema

Add to `api/ws/widgets.schema.yaml` (create if new domain).

### 5. Regenerate Frontend Types

```bash
cd frontend && bun run generate-ws-schemas
```

## Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Domain ID | `{Entity}ID` | `ProjectID`, `UserID` |
| Domain Entity | `{Entity}` | `Project`, `ProjectMember` |
| Domain Enum | `{Entity}{Field}` | `ProjectStatus`, `ProjectMemberRole` |
| Enum Value | `{Enum}{Value}` | `ProjectStatusActive` |
| WS Request | `{Entity}{Action}Request` | `ProjectCreateRequest` |
| WS Response | `{Entity}{Action}Response` | `ProjectCreatedResponse` |
| WS Message Type | `MsgType{Action}` | `MsgTypeProjectCreate` |
| Domain Event | `{Entity}{Action}Event` | `ProjectCreatedEvent` |
| Event Topic | `Topic{Entity}{Action}` | `TopicProjectCreated` |

## Type Relationships

```
                    ┌─────────────────┐
                    │  api/ws/*.yaml  │  (Schema source of truth)
                    └────────┬────────┘
                             │ manual sync
                             ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ internal/domain │◄───│  model/ws/*     │───►│ frontend/model  │
│ (Go types)      │    │ (WS messages)   │    │ (Zod/TS types)  │
└────────┬────────┘    └─────────────────┘    └─────────────────┘
         │
         │ used by
         ▼
┌─────────────────┐    ┌─────────────────┐
│ model/events/*  │───►│ service/*       │
│ (domain events) │    │ (business logic)│
└─────────────────┘    └─────────────────┘
```
